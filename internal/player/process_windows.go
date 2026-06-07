//go:build windows

package player

import "os/exec"

func hideConsoleWindow(cmd *exec.Cmd) {
	_ = cmd
}
