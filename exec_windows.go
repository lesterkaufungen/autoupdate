//go:build windows

package autoupdate

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func execSelf(binary string, args []string) error {
	cmd := exec.Command(binary, args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Start()
	if err != nil {
		return err
	}
	os.Exit(0)
	return nil
}

func runUpdateScript(binary string, args []string, newBinary string, serviceName string) error {
	scriptPath := binary + ".update.ps1"

	err := os.WriteFile(scriptPath, []byte(updateScriptPs1), 0644)
	if err != nil {
		return err
	}

	// PowerShell arguments
	psArgs := []string{
		"-ExecutionPolicy", "Bypass",
		"-NoProfile",
		"-File", scriptPath,
		"-pid", fmt.Sprintf("%d", os.Getpid()),
		"-src", newBinary,
		"-dst", binary,
	}
	if serviceName != "" {
		psArgs = append(psArgs, "-serviceName", serviceName)
	}
	if len(args) > 1 {
		psArgs = append(psArgs, args[1:]...)
	}

	cmd := exec.Command("powershell", psArgs...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: 0x00000008 | 0x00000200, // DETACHED_PROCESS | CREATE_NEW_PROCESS_GROUP
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	os.Exit(0)
	return nil
}
