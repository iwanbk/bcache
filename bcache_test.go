package bcache

import (
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestA(t *testing.T) {
	const (
		key1 = "key1"
		val1 = "val1"
		key2 = "key2"
		val2 = "val2"
	)
	b1, err := New(Config{
		PeerID:     1,
		ListenAddr: "127.0.0.1:12345",
		Peers:      nil,
		MaxKeys:    1000,
		Logger:     logrus.New(),
	})
	require.NoError(t, err)

	b2, err := New(Config{
		PeerID:     2,
		ListenAddr: "127.0.0.1:12346",
		Peers:      []string{"127.0.0.1:12345"},
		MaxKeys:    1000,
		Logger:     logrus.New(),
	})
	require.NoError(t, err)

	// set from b1, and wait in b2
	b1.Set(key1, val1)

	// wait for it to propagate
	time.Sleep(2 * time.Second)

	get, ok := b2.Get(key1)
	require.True(t, ok)
	require.Equal(t, val1, get)

	// set from b2, and wait in b1

	b2.Set(key2, val2)

	// wait for it to propagate
	time.Sleep(2 * time.Second)

	get, ok = b1.Get(key2)
	require.True(t, ok)
	require.Equal(t, val2, get)

}
