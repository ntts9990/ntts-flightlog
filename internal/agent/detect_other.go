//go:build !linux && !darwin && !freebsd && !openbsd && !netbsd && !windows

package agent

// readPPIDComm is a no-op stub for platforms not covered by specific build targets.
func readPPIDComm() string {
	return ""
}
