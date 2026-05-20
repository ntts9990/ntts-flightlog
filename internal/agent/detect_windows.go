//go:build windows

package agent

// readPPIDComm returns "" on Windows; WMIC-based detection is deferred to v2.1.
func readPPIDComm() string {
	return ""
}
