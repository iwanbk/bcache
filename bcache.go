package bcache

import (
	"net"
	"strconv"

	"github.com/weaveworks/mesh"
)

const (
	// weaveworks/mesh channel name
	channel = "bcache"
)

// Bcache represents bcache struct
type Bcache struct {
	peer   *peer
	router *mesh.Router
	logger Logger
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

// Get gets value for the given key
func (b *Bcache) Get(key string) (string, bool) {
	return b.peer.Get(key)
}

// Close closes the cache, free all the resource
func (b *Bcache) Close() error {
	b.logger.Printf("mesh router stopping")
	return b.router.Stop()
}
