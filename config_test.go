package bcache

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	// should be using mac addr
	c := Config{
		ListenAddr: "127.0.0.1:12345",
		Peers:      nil,
		MaxKeys:    1000,
		Logger:     logrus.New(),
	}

	err := c.setDefault()
	require.NoError(t, err)
	require.NotEqual(t, uint64(0), c.PeerID)

	// should be using predefined id
	cfgManual := Config{
		ListenAddr: "127.0.0.1:12345",
		Peers:      nil,
		MaxKeys:    1000,
		Logger:     logrus.New(),
		PeerID:     uint64(2),
	}

	err = cfgManual.setDefault()
	require.NoError(t, err)
	require.Equal(t, uint64(2), cfgManual.PeerID)
}
