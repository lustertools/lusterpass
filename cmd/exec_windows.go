//go:build windows

package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// runExec spawns the target command as a child process with stdio passthrough,
// forwards interrupt signals, and exits with the child's exit code. Windows
// has no execve(2) equivalent, so lusterpass remains alive as the parent for
// the duration of the run.
func runExec(args, envv []string) error {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Env = envv
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting %s: %w", args[0], err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	go func() {
		for sig := range sigCh {
			if cmd.Process != nil {
				_ = cmd.Process.Signal(sig)
			}
		}
	}()

	err := cmd.Wait()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("running %s: %w", args[0], err)
	}
	return nil
}
