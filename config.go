package bcache

import "github.com/weaveworks/mesh"

const (
	defaultDeletionDelay = 100 // default deletion delay : 100 seconds
)

// Config represents bcache configuration
type Config struct {
	// PeerID is unique ID of this bcache
	// if PeerID = 0, it will be set using mac address
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

	// DeletionDelay adds delay before actually delete the key,
	// it is used to handle temporary network connection issue,
	// which could prevent data syncing between nodes.
	// Leave it to 0 make it use default value: 100 seconds.
	DeletionDelay int
}

func (c *Config) setDefault() error {
	// if peerID == 0, get peerID based on the mac address
	if c.PeerID == 0 {
		mac, err := getMacAddress()
		if err != nil {
			return err
		}

		pName, err := mesh.PeerNameFromString(mac)
		if err != nil {
			return err
		}

		c.PeerID = uint64(pName)
	}

	if c.DeletionDelay <= 0 {
		c.DeletionDelay = defaultDeletionDelay
	}

	// if logger is nil, create default nopLogger
	if c.Logger == nil {
		c.Logger = &nopLogger{}
	}

	return nil
}
