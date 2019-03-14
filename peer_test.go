package bcache

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/weaveworks/mesh"
)

func TestPeerOnGossip(t *testing.T) {
	testCases := []struct {
		name    string
		initial map[string]entry
		newMsg  map[string]entry
		delta   map[string]entry
	}{
		{
			name:    "from empty",
			initial: map[string]entry{},
			newMsg: map[string]entry{
				"key2": entry{
					Key:     "key2",
					Val:     "val2",
					Expired: 1,
				},
			},
			delta: map[string]entry{
				"key2": entry{
					Key:     "key2",
					Val:     "val2",
					Expired: 1,
				},
			},
		},
		{
			name: "new different message",
			initial: map[string]entry{
				"key1": entry{
					Key:     "key1",
					Val:     "val1",
					Expired: 1,
				},
			},
			newMsg: map[string]entry{
				"key2": entry{
					Key:     "key2",
					Val:     "val2",
					Expired: 1,
				},
			},
			delta: map[string]entry{
				"key2": entry{
					Key:     "key2",
					Val:     "val2",
					Expired: 1,
				},
			},
		},
		{
			name: "same key diff val",
			initial: map[string]entry{
				"key1": entry{
					Key:     "key1",
					Val:     "val1",
					Expired: 1,
				},
			},
			newMsg: map[string]entry{
				"key1": entry{
					Key:     "key1",
					Val:     "val2",
					Expired: 1,
				},
			},
			delta: map[string]entry{
				"key1": entry{
					Key:     "key1",
					Val:     "val2",
					Expired: 1,
				},
			},
		},
	}

	var (
		peerID1 = mesh.PeerName(1)
		maxKeys = 100
	)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			p, err := newPeer(peerID1, maxKeys, &nopLogger{})
			require.NoError(t, err)

			// initial
			msg := newMessageFromEntries(peerID1, tc.initial)
			buf := msg.Encode()[0]
			_, err = p.OnGossip(buf)
			require.NoError(t, err)

			// newMsg
			newMsg := newMessageFromEntries(peerID1, tc.newMsg)
			buf = newMsg.Encode()[0]
			delta, err := p.OnGossip(buf)
			require.NoError(t, err)

			require.Equal(t, delta.(*message).Entries, tc.delta)
		})
	}
}
