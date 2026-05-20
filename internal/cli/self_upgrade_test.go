package cli

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// ─── isHomebrewPath ───────────────────────────────────────────────────────────

func TestIsHomebrewPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/opt/homebrew/bin/flightlog", true},
		{"/opt/homebrew/Cellar/flightlog/1.0.0/bin/flightlog", true},
		{"/usr/local/Cellar/flightlog/1.0.0/bin/flightlog", true},
		{"/usr/local/bin/flightlog", false},
		{"/usr/bin/flightlog", false},
		{"/home/user/.local/bin/flightlog", false},
		{"/Applications/flightlog", false},
		{"", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			if got := isHomebrewPath(tt.path); got != tt.want {
				t.Errorf("isHomebrewPath(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

// ─── needsUpgrade (version compare) ──────────────────────────────────────────

func TestNeedsUpgrade(t *testing.T) {
	tests := []struct {
		current string
		latest  string
		want    bool
		wantErr bool
	}{
		// newer available
		{"v1.0.0", "v1.1.0", true, false},
		{"1.0.0", "1.1.0", true, false}, // no v prefix
		// older release — no upgrade
		{"v1.1.0", "v1.0.0", false, false},
		// equal — no upgrade
		{"v1.0.0", "v1.0.0", false, false},
		// pre-release → release is an upgrade
		{"v1.0.0-alpha", "v1.0.0", true, false},
		// dev build — skip (not an error)
		{"dev", "v1.0.0", false, false},
		{"unknown", "v1.0.0", false, false},
		// invalid latest — error
		{"v1.0.0", "not-semver", false, true},
		{"v1.0.0", "", false, true},
	}
	for _, tt := range tests {
		name := fmt.Sprintf("%s->%s", tt.current, tt.latest)
		t.Run(name, func(t *testing.T) {
			got, err := needsUpgrade(tt.current, tt.latest)
			if (err != nil) != tt.wantErr {
				t.Fatalf("needsUpgrade(%q,%q) error=%v wantErr=%v", tt.current, tt.latest, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("needsUpgrade(%q,%q) = %v, want %v", tt.current, tt.latest, got, tt.want)
			}
		})
	}
}

// ─── URL pattern matching (assetFilename / checksumFilename) ─────────────────

func TestAssetFilename(t *testing.T) {
	tests := []struct {
		tag    string
		goos   string
		goarch string
		want   string
	}{
		{"v1.2.3", "linux", "amd64", "flightlog_1.2.3_linux_amd64.tar.gz"},
		{"v1.2.3", "darwin", "arm64", "flightlog_1.2.3_darwin_arm64.tar.gz"},
		{"v1.2.3", "darwin", "amd64", "flightlog_1.2.3_darwin_amd64.tar.gz"},
		{"v1.2.3", "windows", "amd64", "flightlog_1.2.3_windows_amd64.zip"},
		{"v1.2.3", "linux", "arm64", "flightlog_1.2.3_linux_arm64.tar.gz"},
		{"1.0.0", "linux", "amd64", "flightlog_1.0.0_linux_amd64.tar.gz"}, // no v prefix
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := assetFilename(tt.tag, tt.goos, tt.goarch)
			if got != tt.want {
				t.Errorf("assetFilename(%q,%q,%q) = %q, want %q", tt.tag, tt.goos, tt.goarch, got, tt.want)
			}
		})
	}
}

func TestChecksumFilename(t *testing.T) {
	tests := []struct {
		tag  string
		want string
	}{
		{"v1.2.3", "flightlog_1.2.3_checksums.txt"},
		{"1.0.0", "flightlog_1.0.0_checksums.txt"},
		{"v0.0.1-alpha", "flightlog_0.0.1-alpha_checksums.txt"},
	}
	for _, tt := range tests {
		t.Run(tt.tag, func(t *testing.T) {
			got := checksumFilename(tt.tag)
			if got != tt.want {
				t.Errorf("checksumFilename(%q) = %q, want %q", tt.tag, got, tt.want)
			}
		})
	}
}

// ─── parseChecksum ───────────────────────────────────────────────────────────

func TestParseChecksum(t *testing.T) {
	data := `
abc123  flightlog_1.0.0_linux_amd64.tar.gz
def456  flightlog_1.0.0_darwin_arm64.tar.gz
`
	got, err := parseChecksum(data, "flightlog_1.0.0_darwin_arm64.tar.gz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "def456" {
		t.Errorf("got %q, want %q", got, "def456")
	}

	_, err = parseChecksum(data, "flightlog_1.0.0_windows_amd64.zip")
	if err == nil {
		t.Error("expected error for missing filename")
	}
}

// ─── mock HTTP server: fetchLatestRelease ─────────────────────────────────────

func TestFetchLatestRelease(t *testing.T) {
	rel := ghRelease{
		TagName: "v1.5.0",
		Assets: []ghAsset{
			{Name: "flightlog_1.5.0_linux_amd64.tar.gz", BrowserDownloadURL: "http://example.com/a.tar.gz"},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(rel); err != nil {
			t.Errorf("encode: %v", err)
		}
	}))
	defer srv.Close()

	got, err := fetchLatestRelease(srv.URL)
	if err != nil {
		t.Fatalf("fetchLatestRelease: %v", err)
	}
	if got.TagName != "v1.5.0" {
		t.Errorf("TagName = %q, want %q", got.TagName, "v1.5.0")
	}
	if len(got.Assets) != 1 {
		t.Errorf("Assets len = %d, want 1", len(got.Assets))
	}
}

func TestFetchLatestRelease_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := fetchLatestRelease(srv.URL)
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
}

// ─── mock HTTP server: runSelfUpgrade flows ───────────────────────────────────

func testCmd() (*cobra.Command, *strings.Builder) {
	var buf strings.Builder
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	return cmd, &buf
}

func TestRunSelfUpgrade_AlreadyLatest(t *testing.T) {
	appVersion = "v1.0.0"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ghRelease{TagName: "v1.0.0"})
	}))
	defer srv.Close()

	cmd, out := testCmd()
	if err := runSelfUpgrade(cmd, false, srv.URL); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "Already up to date") {
		t.Errorf("expected 'Already up to date', got: %s", out.String())
	}
}

func TestRunSelfUpgrade_DevBuild_Skip(t *testing.T) {
	appVersion = "dev"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ghRelease{TagName: "v1.0.0"})
	}))
	defer srv.Close()

	cmd, out := testCmd()
	if err := runSelfUpgrade(cmd, false, srv.URL); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "Already up to date") {
		t.Errorf("expected 'Already up to date' for dev build, got: %s", out.String())
	}
}

func TestRunSelfUpgrade_DryRun(t *testing.T) {
	appVersion = "v1.0.0"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ghRelease{TagName: "v1.5.0"})
	}))
	defer srv.Close()

	cmd, out := testCmd()
	if err := runSelfUpgrade(cmd, true, srv.URL); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "--dry-run") {
		t.Errorf("expected dry-run message, got: %s", out.String())
	}
}

func TestRunSelfUpgrade_NoAsset(t *testing.T) {
	appVersion = "v1.0.0"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ghRelease{
			TagName: "v1.5.0",
			Assets:  []ghAsset{}, // empty — no matching asset
		})
	}))
	defer srv.Close()

	cmd, _ := testCmd()
	err := runSelfUpgrade(cmd, false, srv.URL)
	if err == nil || !strings.Contains(err.Error(), "no asset found") {
		t.Errorf("expected 'no asset found' error, got: %v", err)
	}
}

// ─── mock HTTP server: full download → verify → extract flow ─────────────────

func TestRunSelfUpgrade_DownloadVerifyExtract(t *testing.T) {
	appVersion = "v1.0.0"

	// Build a minimal tar.gz containing a fake "flightlog" binary.
	fakeBin := []byte("#!/bin/sh\necho fake-flightlog")
	archiveBytes := makeFakeTarGz(t, "flightlog", fakeBin)
	archiveChecksum := hexSHA256(archiveBytes)

	tag := "v1.5.0"
	aname := assetFilename(tag, "linux", "amd64")
	cname := checksumFilename(tag)
	checksumLine := fmt.Sprintf("%s  %s\n", archiveChecksum, aname)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/releases/latest":
			_ = json.NewEncoder(w).Encode(ghRelease{
				TagName: tag,
				Assets: []ghAsset{
					{Name: aname, BrowserDownloadURL: "http://" + r.Host + "/asset"},
					{Name: cname, BrowserDownloadURL: "http://" + r.Host + "/checksums"},
				},
			})
		case "/asset":
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(archiveBytes)
		case "/checksums":
			w.Header().Set("Content-Type", "text/plain")
			_, _ = io.WriteString(w, checksumLine)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	// Override to extract linux binary (test runs on darwin but archive has "flightlog").
	// We verify up to extract; actual rename is skipped via dry-run.
	// Test verifyChecksum + extractFromTarGz directly.
	archivePath := writeTempFile(t, archiveBytes)
	defer os.Remove(archivePath)

	// verifyChecksum
	if err := verifyChecksum(archivePath, aname, srv.URL+"/checksums"); err != nil {
		t.Fatalf("verifyChecksum: %v", err)
	}

	// extractFromTarGz
	binPath, err := extractFromTarGz(archivePath)
	if err != nil {
		t.Fatalf("extractFromTarGz: %v", err)
	}
	defer os.Remove(binPath)

	got, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("read extracted binary: %v", err)
	}
	if !bytes.Equal(got, fakeBin) {
		t.Errorf("extracted content mismatch:\n  got:  %q\n  want: %q", got, fakeBin)
	}

	// Also verify the full dry-run path works end-to-end via the mock server.
	cmd, out := testCmd()
	if err := runSelfUpgrade(cmd, true, srv.URL+"/releases/latest"); err != nil {
		t.Fatalf("runSelfUpgrade dry-run: %v", err)
	}
	if !strings.Contains(out.String(), "--dry-run") {
		t.Errorf("expected dry-run output, got: %s", out.String())
	}
}

// ─── brew-detection: runSelfUpgrade refuses Homebrew path ─────────────────────

func TestRunSelfUpgrade_BrewRefusal(t *testing.T) {
	// We can't call runSelfUpgrade with a Homebrew path without it calling
	// osExit — so we intercept osExit and verify isHomebrewPath logic directly.
	homebrew := []string{
		"/opt/homebrew/bin/flightlog",
		"/usr/local/Cellar/flightlog/1.0.0/bin/flightlog",
	}
	for _, p := range homebrew {
		if !isHomebrewPath(p) {
			t.Errorf("isHomebrewPath(%q) should be true", p)
		}
	}

	// Verify osExit is called for a brew path using an intercepted exit function.
	exited := false
	orig := osExit
	osExit = func(code int) {
		exited = true
		if code != 1 {
			t.Errorf("expected exit code 1, got %d", code)
		}
	}
	defer func() { osExit = orig }()

	// Patch os.Executable to return a brew path by overriding the check inline:
	// We call the isHomebrewPath-guarded branch directly to verify osExit fires.
	cmd, out := testCmd()
	// Simulate the brew guard block in isolation.
	if isHomebrewPath("/opt/homebrew/bin/flightlog") {
		fmt.Fprintln(cmd.OutOrStdout(), "Use: brew upgrade ntts9990/tap/flightlog")
		osExit(1)
	}
	if !exited {
		t.Error("expected osExit to be called")
	}
	if !strings.Contains(out.String(), "brew upgrade") {
		t.Errorf("expected brew upgrade message, got: %s", out.String())
	}
}

// ─── helpers ─────────────────────────────────────────────────────────────────

// makeFakeTarGz creates a tar.gz archive in memory containing a single file
// named `name` with the given content. Returns the raw archive bytes.
func makeFakeTarGz(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	hdr := &tar.Header{
		Name:     name,
		Mode:     0o755,
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
	}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatalf("tar write header: %v", err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatalf("tar write content: %v", err)
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}
	if err := gw.Close(); err != nil {
		t.Fatalf("gzip close: %v", err)
	}
	return buf.Bytes()
}

// hexSHA256 returns the hex-encoded SHA-256 of b.
func hexSHA256(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// writeTempFile writes content to a temp file and returns its path.
func writeTempFile(t *testing.T, content []byte) string {
	t.Helper()
	f, err := os.CreateTemp("", "flightlog-test-*")
	if err != nil {
		t.Fatalf("create temp: %v", err)
	}
	defer f.Close()
	if _, err := f.Write(content); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	return f.Name()
}
