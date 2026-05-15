//go:build !windows

package autoupdate

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

func execSelf(binary string, args []string) error {
	return syscall.Exec(binary, args, os.Environ())
}

func runUpdateScript(binary string, args []string, newBinary string, serviceName string) error {
	scriptPath := binary + ".update.sh"

	err := os.WriteFile(scriptPath, []byte(updateScriptSh), 0755)
	if err != nil {
		return err
	}

	// Arguments for the script: PID, SRC, DST, SERVICE_NAME, then original binary args
	cmdArgs := []string{
		fmt.Sprintf("%d", os.Getpid()),
		newBinary,
		binary,
		serviceName,
	}
	if len(args) > 1 {
		cmdArgs = append(cmdArgs, args[1:]...)
	}

	cmd := exec.Command("/bin/sh", append([]string{scriptPath}, cmdArgs...)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	err = cmd.Start()
	if err != nil {
		return err
	}

	os.Exit(0)
	return nil
}
