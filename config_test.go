package bcache

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"github.com/weaveworks/mesh"
)

func TestConfig(t *testing.T) {
	// should be using mac addr
	c := Config{
		ListenAddr: "127.0.0.1:12345",
		Peers:      nil,
		MaxKeys:    1000,
		Logger:     logrus.New(),
	}

	p, err := c.setPeer()
	require.NoError(t, err)
	require.NotEqual(t, mesh.PeerName(uint64(0)), p)

	// should be using predefined id
	cfgManual := Config{
		ListenAddr: "127.0.0.1:12345",
		Peers:      nil,
		MaxKeys:    1000,
		Logger:     logrus.New(),
		PeerID:     uint64(2),
	}

	pMan, err := cfgManual.setPeer()
	require.NoError(t, err)
	require.Equal(t, mesh.PeerName(uint64(2)), pMan)
}
