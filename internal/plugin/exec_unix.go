//go:build unix

package plugin

import (
	"os/exec"
	"syscall"
)

// detachedSysProcAttr puts the child in its own process group so it is isolated
// from the parent's controlling terminal and signals.
func detachedSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}

// killProcessGroup terminates the child AND every process it spawned. Because we
// start the child with Setpgid, the child is the leader of a new process group
// whose id == child pid; signalling -pgid reaches all descendants (so a hung
// `sleep` grandchild can't keep the call blocked past the timeout).
func killProcessGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
