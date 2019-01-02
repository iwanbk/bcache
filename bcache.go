package bcache

import (
	"errors"
	"net"
	"strconv"

	"github.com/weaveworks/mesh"
	"golang.org/x/sync/singleflight"
)

const (
	// weaveworks/mesh channel name
	channel = "bcache"
)

var (
	ErrNilFiller = errors.New("nil filler")
)

// Bcache represents bcache struct
type Bcache struct {
	peer   *peer
	router *mesh.Router
	logger Logger
	flight singleflight.Group
}

// New creates new bcache from the given config
func New(cfg Config) (*Bcache, error) {
	const (
		connLimit = 64 // mesh router connection limit
	)

	var (
		peerName mesh.PeerName
		nickName = cfg.ListenAddr
		logger   = cfg.Logger
	)

	if err := cfg.setDefault(); err != nil {
		return nil, err
	}

	// set peer name
	peerName = mesh.PeerName(cfg.PeerID)

	// if logger is nil, create default nopLogger
	if logger == nil {
		logger = &nopLogger{}
	}

	// parse host port
	host, portStr, err := net.SplitHostPort(cfg.ListenAddr)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, err
	}

	// creates mesh router
	router, err := mesh.NewRouter(mesh.Config{
		Host:               host,
		Port:               port,
		ProtocolMinVersion: mesh.ProtocolMinVersion,
		ConnLimit:          connLimit,
		PeerDiscovery:      true,
		TrustedSubnets:     []*net.IPNet{},
	}, peerName, nickName, mesh.NullOverlay{}, logger)
	if err != nil {
		return nil, err
	}

	// bcache peer
	peer, err := newPeer(peerName, cfg.MaxKeys, logger)
	if err != nil {
		return nil, err
	}

	// creates gossip channel
	gossip, err := router.NewGossip(channel, peer)
	if err != nil {
		return nil, err
	}

	peer.register(gossip)

	// start mesh router
	logger.Printf("mesh router starting at %s", cfg.ListenAddr)
	router.Start()

	// creates new connection to the provided peers
	router.ConnectionMaker.InitiateConnections(cfg.Peers, true)

	return &Bcache{
		peer:   peer,
		router: router,
		logger: logger,
	}, nil
}

// Set sets value for the given key.
//
// expiredTimestamp is unix timestamp when this key will be expired.
// if expiredTimestamp <= 0, the key will never expired
func (b *Bcache) Set(key, val string, expiredTimestamp int64) {
	b.peer.Set(key, val, expiredTimestamp)
}

// Get gets value for the given key.
// It returns the value and true if the key exists
func (b *Bcache) Get(key string) (string, bool) {
	return b.peer.Get(key)
}

// Filler defines func to be called when the given key is not exists
type Filler func(key string) (val string, expired int64, err error)

// GetWithFiller gets value for the given key and fill the cache
// if the given key is not exists
//
// `filler` will be used to fill(Set) the cache
// when the given key is not exist.
//
// It useful to avoid thundering herd of the underlying database
func (b *Bcache) GetWithFiller(key string, filler Filler) (string, bool, error) {
	if filler == nil {
		return "", false, ErrNilFiller
	}

	// get value from cache
	val, ok := b.Get(key)
	if ok {
		return val, ok, nil
	}

	// construct singleflight filler
	flightFn := func() (interface{}, error) {
		val, expired, err := filler(key)
		if err != nil {
			b.logger.Errorf("filler failed: %v", err)
			return nil, err
		}

		// set the key if filler OK
		b.peer.Set(key, val, expired)

		return value{
			value:   val,
			expired: expired,
		}, nil
	}

	// call the filler
	valueIf, err, _ := b.flight.Do(key, func() (interface{}, error) {
		return flightFn()
	})
	if err != nil {
		return "", false, err
	}

	// return the value
	value := valueIf.(value)
	return value.value, true, nil
}

// Close closes the cache, free all the resource
func (b *Bcache) Close() error {
	b.logger.Printf("mesh router stopping")
	return b.router.Stop()
}
