package bcache

import (
	"errors"
	"net"
	"strconv"

	"github.com/weaveworks/mesh"
)

var (
	ErrNilLogger = errors.New("nil logger")
)

const (
	channel = "bcache"
)

type Bcache struct {
	peer *peer
}

type Config struct {
	// PeerID is unique peer ID
	PeerID uint64

	// ListenAddr is listen addr of this bcache peer.
	// used to communicate with other peers
	ListenAddr string

	// Peers is the address of the known peers
	Peers []string

	// MaxKeys defines max number of keys in this cache
	// TODO: implement it
	MaxKeys int

	Logger Logger
}

func (c Config) validate() error {
	if c.Logger == nil {
		return ErrNilLogger
	}
	return nil
}
func New(cfg Config) (*Bcache, error) {
	const (
		connLimit = 64
	)

	var (
		peerName = mesh.PeerName(cfg.PeerID)
		nickName = cfg.ListenAddr
		log      = cfg.Logger
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
	}, peerName, nickName, mesh.NullOverlay{}, log)
	if err != nil {
		return nil, err
	}

	// bcache peer
	peer, err := newPeer(peerName, cfg.MaxKeys)
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
	log.Printf("mesh router starting at %s", cfg.ListenAddr)
	router.Start()

	defer func() {
		log.Printf("mesh router stopping")
		router.Stop()
	}()

	// creates new connection to the provided peers
	router.ConnectionMaker.InitiateConnections(cfg.Peers, true)

	return &Bcache{
		peer: peer,
	}, nil
}

func (b *Bcache) Set(key, val interface{}) {
	b.peer.Set(key, val)
}
func (b *Bcache) Get(key interface{}) (interface{}, bool) {
	return b.peer.Get(key)
}
