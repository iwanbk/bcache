package bcache

import "github.com/weaveworks/mesh"

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

func (c *Config) setPeer() (mesh.PeerName, error) {
	if c.PeerID != uint64(0) {
		return mesh.PeerName(c.PeerID), nil
	}

	mac, err := getMacAddress()
	if err != nil {
		return mesh.PeerName(0), err
	}

	pName, err := mesh.PeerNameFromString(mac)
	if err != nil {
		return mesh.PeerName(0), err
	}

	return pName, nil
}
