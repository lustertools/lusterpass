//go:build unix

package cmd

import (
	"fmt"
	"os/exec"
	"syscall"
)

// runExec replaces the current lusterpass process with the target command via
// execve(2). On success, this call does not return. On failure, the process
// remains lusterpass and the error is propagated to the user.
func runExec(args, envv []string) error {
	path, err := exec.LookPath(args[0])
	if err != nil {
		return fmt.Errorf("command not found: %s", args[0])
	}

	if err := syscall.Exec(path, args, envv); err != nil {
		return fmt.Errorf("exec %s: %w", path, err)
	}

	// Unreachable on success — execve replaces the process image.
	return nil
}
