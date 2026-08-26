package cli

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/mod/semver"
)

const upgradeAPIURL = "https://api.github.com/repos/ntts9990/ntts-flightlog/releases/latest"

// ghRelease is the subset of the GitHub releases API response we need.
type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// osExit is overridable in tests to avoid calling os.Exit directly.
var osExit = os.Exit

func newSelfUpgradeCmd() *cobra.Command {
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "self-upgrade",
		Short: "Upgrade flightlog to the latest GitHub release",
		Long: `Downloads and installs the latest flightlog release from GitHub.

The archive is verified via SHA-256 before the binary is replaced atomically.
If you installed via Homebrew, use 'brew upgrade ntts9990/tap/flightlog' instead.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSelfUpgrade(cmd, dryRun, upgradeAPIURL)
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Check for updates without applying them")
	return cmd
}

// runSelfUpgrade implements the upgrade logic. apiURL is injectable for tests.
func runSelfUpgrade(cmd *cobra.Command, dryRun bool, apiURL string) error {
	// 1. Refuse Homebrew installs immediately.
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("self-upgrade: resolve executable: %w", err)
	}
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("self-upgrade: eval symlinks: %w", err)
	}
	if isHomebrewPath(execPath) {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Use: brew upgrade ntts9990/tap/flightlog")
		osExit(1)
		return nil // unreachable; silences static analysis
	}

	// 2. Fetch latest release metadata.
	rel, err := fetchLatestRelease(apiURL)
	if err != nil {
		return fmt.Errorf("self-upgrade: fetch release: %w", err)
	}

	// 3. Semver compare.
	upgrade, err := needsUpgrade(appVersion, rel.TagName)
	if err != nil {
		return fmt.Errorf("self-upgrade: version check: %w", err)
	}
	cmd.Printf("current: %s  latest: %s\n", appVersion, rel.TagName)
	if !upgrade {
		cmd.Println("Already up to date.")
		return nil
	}
	cmd.Printf("New version available: %s\n", rel.TagName)
	if dryRun {
		cmd.Println("--dry-run: no changes applied.")
		return nil
	}

	// 4. Resolve asset and checksum URLs from the release asset list.
	aname := assetFilename(rel.TagName, runtime.GOOS, runtime.GOARCH)
	cname := checksumFilename(rel.TagName)
	var assetURL, csumURL string
	for _, a := range rel.Assets {
		switch a.Name {
		case aname:
			assetURL = a.BrowserDownloadURL
		case cname:
			csumURL = a.BrowserDownloadURL
		}
	}
	if assetURL == "" {
		return fmt.Errorf("self-upgrade: no asset found for %s/%s (expected %q)", runtime.GOOS, runtime.GOARCH, aname)
	}
	if csumURL == "" {
		return fmt.Errorf("self-upgrade: no checksums file found (expected %q)", cname)
	}

	// 5. Download archive to a temp file.
	cmd.Printf("Downloading %s...\n", aname)
	archivePath, err := downloadToTemp(assetURL)
	if err != nil {
		return fmt.Errorf("self-upgrade: download: %w", err)
	}
	defer func() { _ = os.Remove(archivePath) }()

	// 6. Verify SHA-256 checksum.
	cmd.Println("Verifying checksum...")
	if err := verifyChecksum(archivePath, aname, csumURL); err != nil {
		return fmt.Errorf("self-upgrade: %w", err)
	}

	// 7. Extract the binary from the archive.
	binPath, err := extractBinary(archivePath, runtime.GOOS)
	if err != nil {
		return fmt.Errorf("self-upgrade: extract: %w", err)
	}
	defer func() { _ = os.Remove(binPath) }()

	if err := os.Chmod(binPath, 0o755); err != nil {
		return fmt.Errorf("self-upgrade: chmod: %w", err)
	}

	// 8. Atomic replace: rename temp binary over the current executable.
	cmd.Printf("Installing %s...\n", rel.TagName)
	if err := os.Rename(binPath, execPath); err != nil {
		return fmt.Errorf("self-upgrade: replace binary: %w", err)
	}
	cmd.Printf("Successfully upgraded to %s.\n", rel.TagName)
	return nil
}

// isHomebrewPath reports whether the executable path is under a Homebrew-managed prefix.
func isHomebrewPath(path string) bool {
	for _, prefix := range []string{"/opt/homebrew/", "/usr/local/Cellar/"} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// needsUpgrade returns true when latest is strictly newer than current.
// A non-semver current (e.g. "dev") is treated as "skip upgrade" without error.
// A non-semver latest is always an error.
func needsUpgrade(current, latest string) (bool, error) {
	lat := ensureV(latest)
	if !semver.IsValid(lat) {
		return false, fmt.Errorf("latest tag %q is not valid semver", latest)
	}
	cur := ensureV(current)
	if !semver.IsValid(cur) {
		// Dev / dirty build — cannot compare; skip upgrade silently.
		return false, nil
	}
	return semver.Compare(lat, cur) > 0, nil
}

func ensureV(s string) string {
	if strings.HasPrefix(s, "v") {
		return s
	}
	return "v" + s
}

// assetFilename returns the GoReleaser archive name for the given release tag,
// OS, and architecture, e.g. "flightlog_1.2.3_linux_amd64.tar.gz".
func assetFilename(tag, goos, goarch string) string {
	ver := strings.TrimPrefix(tag, "v")
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("flightlog_%s_%s_%s%s", ver, goos, goarch, ext)
}

// checksumFilename returns the GoReleaser checksums file name for a tag,
// e.g. "flightlog_1.2.3_checksums.txt".
func checksumFilename(tag string) string {
	ver := strings.TrimPrefix(tag, "v")
	return fmt.Sprintf("flightlog_%s_checksums.txt", ver)
}

// fetchLatestRelease calls the GitHub Releases API and returns the parsed response.
func fetchLatestRelease(apiURL string) (*ghRelease, error) {
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}
	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decode release JSON: %w", err)
	}
	if rel.TagName == "" {
		return nil, fmt.Errorf("release has no tag_name")
	}
	return &rel, nil
}

// downloadToTemp downloads url into a temporary file and returns its path.
// The caller is responsible for removing the file.
func downloadToTemp(url string) (string, error) {
	resp, err := http.Get(url) //nolint:gosec // URL comes from the GitHub API response
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download %s: HTTP %d", url, resp.StatusCode)
	}

	f, err := os.CreateTemp("", "flightlog-upgrade-*")
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// verifyChecksum downloads the checksums file at checksumURL, locates the
// expected SHA-256 for filename, and compares it against the file at filePath.
func verifyChecksum(filePath, filename, checksumURL string) error {
	resp, err := http.Get(checksumURL) //nolint:gosec
	if err != nil {
		return fmt.Errorf("fetch checksums: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch checksums: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read checksums: %w", err)
	}

	expected, err := parseChecksum(string(data), filename)
	if err != nil {
		return err
	}
	actual, err := sha256File(filePath)
	if err != nil {
		return err
	}
	if actual != expected {
		return fmt.Errorf("checksum mismatch for %s:\n  got:  %s\n  want: %s", filename, actual, expected)
	}
	return nil
}

// parseChecksum finds the hex SHA-256 for filename in the GoReleaser checksums
// blob. Lines have the BSD shasum format: "<hex>  <filename>".
func parseChecksum(data, filename string) (string, error) {
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		if fields[1] == filename {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("checksum not found for %q in checksums file", filename)
}

// sha256File returns the hex-encoded SHA-256 digest of a file.
func sha256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// extractBinary extracts the flightlog binary from a tar.gz (unix) or zip
// (windows) archive into a temporary file and returns its path.
func extractBinary(archivePath, goos string) (string, error) {
	if goos == "windows" {
		return extractFromZip(archivePath)
	}
	return extractFromTarGz(archivePath)
}

func extractFromTarGz(archivePath string) (string, error) {
	af, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer func() { _ = af.Close() }()

	gr, err := gzip.NewReader(af)
	if err != nil {
		return "", fmt.Errorf("gzip open: %w", err)
	}
	defer func() { _ = gr.Close() }()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("tar: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if filepath.Base(hdr.Name) != "flightlog" {
			continue
		}
		out, err := os.CreateTemp("", "flightlog-bin-*")
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(out, tr); err != nil {
			_ = out.Close()
			_ = os.Remove(out.Name())
			return "", err
		}
		_ = out.Close()
		return out.Name(), nil
	}
	return "", fmt.Errorf("binary 'flightlog' not found in archive")
}

func extractFromZip(archivePath string) (string, error) {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("zip open: %w", err)
	}
	defer func() { _ = zr.Close() }()

	for _, f := range zr.File {
		if filepath.Base(f.Name) != "flightlog.exe" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		out, err := os.CreateTemp("", "flightlog-bin-*.exe")
		if err != nil {
			_ = rc.Close()
			return "", err
		}
		if _, err := io.Copy(out, rc); err != nil {
			_ = rc.Close()
			_ = out.Close()
			_ = os.Remove(out.Name())
			return "", err
		}
		_ = rc.Close()
		_ = out.Close()
		return out.Name(), nil
	}
	return "", fmt.Errorf("binary 'flightlog.exe' not found in zip")
}
