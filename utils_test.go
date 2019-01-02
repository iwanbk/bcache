package bcache

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetMacAddress(t *testing.T) {
	mac, err := getMacAddress()
	require.NoError(t, err)
	require.NotEqual(t, "", mac)
}
