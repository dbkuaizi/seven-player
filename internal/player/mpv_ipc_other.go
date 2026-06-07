//go:build !windows

package player

import (
	"errors"
	"net"
	"time"
)

func newMPVIPCServerName() string {
	return ""
}

func dialMPVIPC(_ string, _ time.Duration) (net.Conn, error) {
	return nil, errors.New("MPV IPC is only enabled on Windows")
}
