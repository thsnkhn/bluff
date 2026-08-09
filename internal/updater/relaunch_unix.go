//go:build !windows

package updater

import (
	"fmt"
	"os"
	"syscall"
)

// Relaunch replaces the current process after Bubble Tea has restored the
// terminal. Calling this before Program.Run returns can corrupt raw mode.
func Relaunch() error {
	executable, err := updatedExecutable()
	if err != nil {
		return err
	}
	args := append([]string{executable}, os.Args[1:]...)
	if err := syscall.Exec(executable, args, os.Environ()); err != nil {
		return fmt.Errorf("restart Bluff: %w", err)
	}
	return nil
}
