//go:build !windows

package app

import (
	"fmt"
	"os/exec"
)

// launchDetachedInstaller runs fallback installer launch on non-Windows platforms.
func launchDetachedInstaller(installerPath string) error {
	cmd := exec.Command("open", installerPath)
	if err := cmd.Start(); err != nil {
		cmd = exec.Command("xdg-open", installerPath)
		return cmd.Start()
	}
	return nil
}
