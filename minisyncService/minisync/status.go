package minisync

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// ServiceStatus represents the status of a service as a string.
type ServiceStatus string

const (
	// StatusNotInstalled indicates that the service is not installed on the system.
	StatusNotInstalled ServiceStatus = "NotInstalled"

	// StatusStopped indicates that the service is installed but currently stopped.
	StatusStopped ServiceStatus = "Stopped"

	// StatusRunning indicates that the service is currently running.
	StatusRunning ServiceStatus = "Running"

	// StatusPaused indicates that the service is currently paused.
	StatusPaused ServiceStatus = "Paused"
)

// GetServiceStatus checks the status of a Windows service by executing the `sc query` command
// and parsing its output. It returns a string representing the service's status, which can be
// one of the predefined statuses (Running, Stopped, Paused, or NotInstalled).
func GetServiceStatus(serviceName string) (string, error) {
	cmd := exec.Command("sc", "query", serviceName)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err != nil {
		if strings.Contains(out.String(), "The specified service does not exist as an installed service.") {
			return string(StatusNotInstalled), nil
		}
		return "", err
	}

	output := out.String()
	if strings.Contains(output, "STATE              : 4  RUNNING") {
		return string(StatusRunning), nil
	} else if strings.Contains(output, "STATE              : 1  STOPPED") {
		return string(StatusStopped), nil
	} else if strings.Contains(output, "STATE              : 7  PAUSED") {
		return string(StatusPaused), nil
	}

	return "", fmt.Errorf("unable to determine service status")
}
