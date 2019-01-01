package bcache

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

// Two nodes
// peer 2 could read what peer 1 write
func TestSimple(t *testing.T) {
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
	defer b1.Close()

	b2, err := New(Config{
		PeerID:     2,
		ListenAddr: "127.0.0.1:12346",
		Peers:      []string{"127.0.0.1:12345"},
		MaxKeys:    1000,
		Logger:     logrus.New(),
	})
	require.NoError(t, err)
	defer b2.Close()

	// set from b1, and wait in b2
	b1.Set(key1, val1, 0)

	// wait for it to propagate
	time.Sleep(2 * time.Second)

	get, ok := b2.Get(key1)
	require.True(t, ok)
	require.Equal(t, val1, get)

	// set from b2, and wait in b1

	b2.Set(key2, val2, 0)

	// wait for it to propagate
	time.Sleep(2 * time.Second)

	get, ok = b1.Get(key2)
	require.True(t, ok)
	require.Equal(t, val2, get)

}

// Second peer join after first peer set the keys
func TestJoinLater(t *testing.T) {
	const (
		numKeys = 15
	)
	var (
		keyvals = make(map[string]string)
	)
	for i := 0; i < numKeys; i++ {
		k := fmt.Sprintf("key_%d", i)
		v := fmt.Sprintf("val_%d", i)
		keyvals[k] = v
	}

	// create first node
	b1, err := New(Config{
		PeerID:     1,
		ListenAddr: "127.0.0.1:12347",
		Peers:      nil,
		MaxKeys:    1000,
		Logger:     logrus.New(),
	})
	require.NoError(t, err)
	defer b1.Close()

	// set values
	for k, v := range keyvals {
		b1.Set(k, v, 0)
	}

	b2, err := New(Config{
		PeerID:     2,
		ListenAddr: "127.0.0.1:12348",
		Peers:      []string{"127.0.0.1:12347"},
		MaxKeys:    1000,
		Logger:     logrus.New(),
	})
	require.NoError(t, err)
	defer b2.Close()

	// wait for it to propagate
	time.Sleep(2 * time.Second)

	// check we could get it from b2
	for k, v := range keyvals {
		got, ok := b2.Get(k)
		require.True(t, ok)
		require.Equal(t, v, got)
	}
}

func TestFiller(t *testing.T) {
	var (
		errFillerFailed = errors.New("filler failed")
	)

	testCases := []struct {
		name   string
		filler Filler
		err    error
		key    string
		ok     bool
	}{
		{
			name:   "nil filler",
			filler: nil,
			err:    ErrNilFiller,
			key:    "nil",
		},
		{
			name: "valid filler",
			filler: func(key string) (string, int64, error) {
				return key, 0, nil
			},
			key: "valid",
			err: nil,
			ok:  true,
		},
		{
			name: "failed filler",
			filler: func(key string) (string, int64, error) {
				return "", 0, errFillerFailed
			},
			key: "failed",
			err: errFillerFailed,
		},
	}

	bc, err := New(Config{
		PeerID:     2,
		ListenAddr: "127.0.0.1:12349",
		MaxKeys:    1000,
		Logger:     logrus.New(),
	})
	require.NoError(t, err)
	defer bc.Close()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			val, ok, err := bc.GetWithFiller(tc.key, tc.filler)
			require.Equal(t, tc.err, err)
			if tc.err != nil {
				return
			}
			require.Equal(t, tc.key, val)
			require.Equal(t, tc.ok, ok)
		})
	}
}
