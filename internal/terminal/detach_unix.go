//go:build !windows

package terminal

import (
	"os/exec"
	"syscall"
)

// detach puts the command in a session of its own, so closing the terminal
// the app runs in does not close it
func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
