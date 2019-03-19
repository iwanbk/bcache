package bcache

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/weaveworks/mesh"
)

func TestMessageMerge(t *testing.T) {
	testCases := []struct {
		name     string
		initial  map[string]entry
		other    map[string]entry
		complete map[string]entry
	}{
		{
			name:    "from empty",
			initial: map[string]entry{},
			other: map[string]entry{
				"key2": entry{
					Key:     "key2",
					Val:     "val2",
					Expired: 1,
				},
			},
			complete: map[string]entry{
				"key2": entry{
					Key:     "key2",
					Val:     "val2",
					Expired: 1,
				},
			},
		},
		{
			name: "new key",
			initial: map[string]entry{
				"key1": entry{
					Key:     "key1",
					Val:     "val1",
					Expired: 1,
				},
			},
			other: map[string]entry{
				"key2": entry{
					Key:     "key2",
					Val:     "val2",
					Expired: 1,
				},
			},
			complete: map[string]entry{
				"key1": entry{
					Key:     "key1",
					Val:     "val1",
					Expired: 1,
				},
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
			other: map[string]entry{
				"key1": entry{
					Key:     "key1",
					Val:     "val2",
					Expired: 2,
				},
			},
			complete: map[string]entry{
				"key1": entry{
					Key:     "key1",
					Val:     "val2",
					Expired: 2,
				},
			},
		},
		{
			name: "same key same val",
			initial: map[string]entry{
				"key1": entry{
					Key:     "key1",
					Val:     "val1",
					Expired: 1,
				},
			},
			other: map[string]entry{
				"key1": entry{
					Key:     "key1",
					Val:     "val1",
					Expired: 1,
				},
			},
			complete: map[string]entry{
				"key1": entry{
					Key:     "key1",
					Val:     "val1",
					Expired: 1,
				},
			},
		},
		{
			name: "same key dif val same exp",
			initial: map[string]entry{
				"key1": entry{
					Key:     "key1",
					Val:     "val1",
					Expired: 1,
				},
			},
			other: map[string]entry{
				"key1": entry{
					Key:     "key1",
					Val:     "val2",
					Expired: 1,
				},
			},
			complete: map[string]entry{
				"key1": entry{
					Key:     "key1",
					Val:     "val1",
					Expired: 1,
				},
			},
		},
	}

	var (
		peerID1 = mesh.PeerName(1)
	)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// initial
			msg := newMessageFromEntries(peerID1, tc.initial)

			// other
			newMsg := newMessageFromEntries(peerID1, tc.other)

			complete := msg.Merge(newMsg)

			require.Equal(t, tc.complete, complete.(*message).Entries)
		})
	}
}
