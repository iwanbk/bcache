package bcache

import (
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/weaveworks/mesh"
	"golang.org/x/sync/singleflight"
)

const (
	// weaveworks/mesh channel name
	channel = "bcache"
)

var (
	// ErrNilFiller returned when GetWithFiller called with nil
	// filler func
	ErrNilFiller = errors.New("nil filler")
)

// Bcache represents bcache struct
type Bcache struct {
	peer          *peer
	router        *mesh.Router
	logger        Logger
	flight        singleflight.Group
	deletionDelay time.Duration
}

// New creates new bcache from the given config
func New(cfg Config) (*Bcache, error) {
	const (
		connLimit = 64 // mesh router connection limit
	)

	if err := cfg.setDefault(); err != nil {
		return nil, err
	}

	var (
		peerName = mesh.PeerName(cfg.PeerID)
		nickName = cfg.ListenAddr
		logger   = cfg.Logger
	)

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
		peer:          peer,
		router:        router,
		logger:        logger,
		deletionDelay: time.Duration(cfg.DeletionDelay) * time.Second,
	}, nil
}

// Set sets value for the given key with the given ttl in second.
// if ttl <= 0, the key will expired instantly
func (b *Bcache) Set(key, val string, ttl int) {
	if ttl <= 0 {
		b.Delete(key)
		return
	}
	b.set(key, val, ttl)
}

func (b *Bcache) set(key, val string, ttl int) int64 {
	expired := time.Now().Add(time.Duration(ttl) * time.Second).UnixNano()
	b.peer.Set(key, val, expired)
	return expired
}

// Get gets value for the given key.
//
// It returns the value and true if the key exists
func (b *Bcache) Get(key string) (string, bool) {
	return b.peer.Get(key)
}

// Delete the given key.
//
func (b *Bcache) Delete(key string) {
	deleteTs := time.Now().Add(b.deletionDelay).UnixNano()
	b.peer.Delete(key, deleteTs)
}

// Filler defines func to be called when the given key is not exists
type Filler func(key string) (val string, err error)

// GetWithFiller gets value for the given key and fill the cache
// if the given key is not exists.
//
// `filler` will be used to fill(Set) the cache
// when the given key is not exist.
// Even there are many goroutines which call `GetWithFiller`, the given `Filler` func
// will only called once for each of the key.
//
//
// It useful to avoid cache stampede to  the underlying database
func (b *Bcache) GetWithFiller(key string, filler Filler, ttl int) (string, error) {
	if filler == nil {
		return "", ErrNilFiller
	}

	// get value from cache
	val, ok := b.Get(key)
	if ok {
		return val, nil
	}

	// construct singleflight filler
	flightFn := func() (interface{}, error) {
		val, err := filler(key)
		if err != nil {
			b.logger.Errorf("filler failed: %v", err)
			return nil, err
		}

		expired := b.set(key, val, ttl)

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
		return "", err
	}

	// return the value
	value := valueIf.(value)
	return value.value, nil
}

// Close closes the cache, free all the resource
func (b *Bcache) Close() error {
	b.logger.Printf("mesh router stopping")
	return b.router.Stop()
}
