//go:build !windows

package player

import "os/exec"

func hideConsoleWindow(_ *exec.Cmd) {}
