//go:build windows

package app

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// launchDetachedInstaller launches the downloaded installer in a detached process group
// that waits for the parent process to exit, runs the installer silently, and cleans up the installer binary.
func launchDetachedInstaller(installerPath string) error {
	pid := os.Getpid()

	// PowerShell script executed in background:
	// 1. Wait for parent process PID to terminate so files are unlocked.
	// 2. Run the installer (wait for finish).
	// 3. Remove the temporary installer file.
	script := fmt.Sprintf(`$proc = Get-Process -Id %d -ErrorAction SilentlyContinue; if ($proc) { $proc.WaitForExit(15000) }; Start-Process -FilePath '%s' -ArgumentList '/SILENT /CLOSEAPPLICATIONS /RESTARTAPPLICATIONS' -Wait; Start-Sleep -Seconds 2; Remove-Item -Path '%s' -Force -ErrorAction SilentlyContinue`,
		pid, installerPath, installerPath)

	cmd := exec.Command("powershell.exe", "-NoProfile", "-WindowStyle", "Hidden", "-Command", script)
	// CREATE_NEW_PROCESS_GROUP = 0x00000200, DETACHED_PROCESS = 0x00000008, CREATE_NO_WINDOW = 0x08000000
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x08000000 | 0x00000200,
	}

	return cmd.Start()
}
