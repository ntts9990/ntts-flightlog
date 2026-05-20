//go:build darwin || freebsd || openbsd || netbsd

package agent

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// readPPIDComm queries the parent process command via ps(1).
func readPPIDComm() string {
	ppid := os.Getppid()
	out, err := exec.Command("ps", "-o", "command=", "-p", strconv.Itoa(ppid)).Output()
	if err != nil {
		return ""
	}
	comm := strings.TrimSpace(string(out))
	// Strip path prefix: /Applications/Claude.app/Contents/MacOS/Claude → Claude
	if idx := strings.LastIndexByte(comm, '/'); idx >= 0 {
		comm = comm[idx+1:]
	}
	// Strip arguments: "claude-desktop --flag" → "claude-desktop"
	if idx := strings.IndexByte(comm, ' '); idx >= 0 {
		comm = comm[:idx]
	}
	return comm
}
