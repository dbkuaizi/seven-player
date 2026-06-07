//go:build windows

package player

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/Microsoft/go-winio"
)

func newMPVIPCServerName() string {
	return fmt.Sprintf(`\\.\pipe\seven-player-mpv-%d-%d`, os.Getpid(), time.Now().UnixNano())
}

func dialMPVIPC(server string, timeout time.Duration) (net.Conn, error) {
	return winio.DialPipe(server, &timeout)
}
