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

// Config represents bcache configuration
type Config struct {
	// PeerID is unique ID of this bcache
	PeerID uint64

	// ListenAddr is listen addr of this bcache peer.
	// used to communicate with other peers
	ListenAddr string

	// Peers is the address of the known peers.
	// we don't need to know all of the other peers,
	// gossip protocol will find the other peers
	Peers []string

	// MaxKeys defines max number of keys in this cache
	MaxKeys int

	// Logger to be used
	// leave it nil to use default logger which do nothing
	Logger Logger
}

// New creates new bcache from the given config
func New(cfg Config) (*Bcache, error) {
	const (
		connLimit = 64 // mesh router connection limit
	)

	var (
		peerName = mesh.PeerName(cfg.PeerID)
		nickName = cfg.ListenAddr
		logger   = cfg.Logger
	)

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
func (b *Bcache) Get(key string) (interface{}, bool) {
	return b.peer.Get(key)
}

// Close closes the cache, free all the resource
func (b *Bcache) Close() error {
	b.logger.Printf("mesh router stopping")
	return b.router.Stop()
}
