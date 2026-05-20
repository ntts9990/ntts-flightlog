//go:build linux

package agent

import (
	"fmt"
	"os"
	"strings"
)

// readPPIDComm reads the command name of the parent process from /proc.
func readPPIDComm() string {
	ppid := os.Getppid()
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", ppid))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
