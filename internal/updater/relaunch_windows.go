//go:build windows

package updater

import "errors"

func Relaunch() error {
	return errors.New("automatic restart is not supported on Windows yet")
}
