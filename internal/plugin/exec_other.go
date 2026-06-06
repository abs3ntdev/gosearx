//go:build !unix

package plugin

import (
	"os/exec"
	"syscall"
)

// detachedSysProcAttr is a no-op on non-Unix platforms.
func detachedSysProcAttr() *syscall.SysProcAttr { return nil }

// killProcessGroup falls back to killing just the child on non-Unix platforms.
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
