//go:build !unix

package execengine

import (
	"os/exec"
	"syscall"
)

func detachedSysProcAttr() *syscall.SysProcAttr { return nil }

func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
