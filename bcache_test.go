package bcache

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// Three nodes
// - peer 2 & 3 could read what peer 1 write
// - peer 2 could update that value
// - delete propagated to other peers
func TestIntegration(t *testing.T) {
	const (
		val1    = "val1"
		val2    = "val2"
		numKeys = 50
	)
	var (
		keys      []string
		expiredIn = 10 * time.Minute
		waitDur   = 3 * time.Second
	)

	b1, err := New(Config{
		PeerID:     1,
		ListenAddr: "127.0.0.1:12345",
		Peers:      nil,
		MaxKeys:    1000,
		Logger:     &nopLogger{},
	})
	require.NoError(t, err)
	defer b1.Close()

	b2, err := New(Config{
		PeerID:     2,
		ListenAddr: "127.0.0.1:12346",
		Peers:      []string{"127.0.0.1:12345"},
		MaxKeys:    1000,
		Logger:     &nopLogger{},
	})
	require.NoError(t, err)
	defer b2.Close()

	b3, err := New(Config{
		PeerID:     3,
		ListenAddr: "127.0.0.1:12356",
		Peers:      []string{"127.0.0.1:12345"},
		MaxKeys:    1000,
		Logger:     &nopLogger{},
	})
	require.NoError(t, err)
	defer b3.Close()

	// generate keys
	for i := 0; i < numKeys; i++ {
		keys = append(keys, fmt.Sprintf("key-%d", i))
	}

	// ---------- set from b1, check in b2 & b3 -----------
	for _, key := range keys {
		b1.Set(key, val1, time.Now().Add(expiredIn).UnixNano())
	}

	// wait for it to propagate and check from b2 b3
	time.Sleep(waitDur)

	for _, key := range keys {
		get, exp, ok := b2.Get(key)
		require.True(t, ok)
		require.NotZero(t, exp)
		require.Equal(t, val1, get)

		get, exp, ok = b3.Get(key)
		require.True(t, ok)
		require.NotZero(t, exp)
		require.Equal(t, val1, get)

	}

	// ----------- set from b2, check in b1 & b3 --------------
	for _, key := range keys {
		b2.Set(key, val2, time.Now().Add(expiredIn).UnixNano())
	}

	// wait for it to propagate and check from b1 & b3
	time.Sleep(waitDur)

	for _, key := range keys {
		get, exp, ok := b1.Get(key)
		require.True(t, ok)
		require.NotZero(t, exp)
		require.Equal(t, val2, get)

		get, exp, ok = b3.Get(key)
		require.True(t, ok)
		require.NotZero(t, exp)
		require.Equal(t, val2, get)

	}

	// ------ delete from b1, and check  b2 & b3 ----------
	for _, key := range keys {
		b1.Delete(key, time.Now().Add(expiredIn).UnixNano())
	}
	// wait for it to propagate and check from b2
	time.Sleep(waitDur)

	for _, key := range keys {
		_, _, exists := b2.Get(key)
		require.False(t, exists)

		_, _, exists = b3.Get(key)
		require.False(t, exists)
	}

}

// Second peer join after first peer set the keys
func TestJoinLater(t *testing.T) {
	const (
		numKeys = 15
	)
	var (
		keyvals   = make(map[string]string)
		expiredIn = 10 * time.Minute
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
		Logger:     &nopLogger{},
	})
	require.NoError(t, err)
	defer b1.Close()

	// set values
	for k, v := range keyvals {
		b1.Set(k, v, time.Now().Add(expiredIn).Unix())
	}

	b2, err := New(Config{
		PeerID:     2,
		ListenAddr: "127.0.0.1:12348",
		Peers:      []string{"127.0.0.1:12347"},
		MaxKeys:    1000,
		Logger:     &nopLogger{},
	})
	require.NoError(t, err)
	defer b2.Close()

	// wait for it to propagate
	time.Sleep(2 * time.Second)

	// check we could get it from b2
	for k, v := range keyvals {
		got, _, ok := b2.Get(k)
		require.True(t, ok)
		require.Equal(t, v, got)
	}
}

func TestFiller(t *testing.T) {
	var (
		errFillerFailed = errors.New("filler failed")
		expiredIn       = 10 * time.Minute
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
				return key, time.Now().Add(expiredIn).Unix(), nil
			},
			key: "valid",
			err: nil,
			ok:  true,
		},
		{
			name: "failed filler",
			filler: func(key string) (string, int64, error) {
				return "", time.Now().Add(expiredIn).Unix(), errFillerFailed
			},
			key: "failed",
			err: errFillerFailed,
		},
	}

	bc, err := New(Config{
		PeerID:     2,
		ListenAddr: "127.0.0.1:12349",
		MaxKeys:    1000,
		Logger:     &nopLogger{},
	})
	require.NoError(t, err)
	defer bc.Close()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			val, exp, err := bc.GetWithFiller(tc.key, tc.filler)
			require.Equal(t, tc.err, err)
			if tc.err != nil {
				return
			}
			require.Equal(t, tc.key, val)
			require.NotZero(t, exp)
		})
	}
}

func TestValidationError(t *testing.T) {
	// invalid address because no port
	c := Config{
		PeerID:     1,
		ListenAddr: "127.0.0.1",
		Peers:      nil,
		MaxKeys:    1000,
		Logger:     &nopLogger{},
	}

	_, err := New(c)
	require.Error(t, err)

	// invalid address because wrong port
	c.ListenAddr = "127.0.0.1:abcd"
	_, err = New(c)
	require.Error(t, err)

	// null logger not cause error
	c.Logger = nil
	c.ListenAddr = "127.0.0.1:12666"

	b1, err := New(c)
	require.NoError(t, err)
	require.NotNil(t, b1)
	defer b1.Close()
}
