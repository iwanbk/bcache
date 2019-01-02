package bcache

import (
	"errors"
	"net"
)

var (
	errMacAddressNotFound = errors.New("mac address not found in this machine")
)

// getMacAddress get mac address of this machine
func getMacAddress() (string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, intf := range interfaces {
		if intf.Flags&net.FlagUp != 0 && intf.HardwareAddr != nil {
			return intf.HardwareAddr.String(), nil
		}
	}

	return "", errMacAddressNotFound
}
