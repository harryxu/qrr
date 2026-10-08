// Package update installs verified GitHub releases over the current executable.
package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

const (
	latestURL       = "https://api.github.com/repos/harryxu/qrr/releases/latest"
	maxMetadataSize = 1 << 20
	maxArchiveSize  = 128 << 20
	maxBinarySize   = 128 << 20
)

type asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type release struct {
	Tag        string  `json:"tag_name"`
	Draft      bool    `json:"draft"`
	Prerelease bool    `json:"prerelease"`
	Assets     []asset `json:"assets"`
}

type updater struct {
	client     *http.Client
	latestURL  string
	os, arch   string
	executable func() (string, error)
}

// Upgrade replaces the running executable only when a newer stable release exists.
// Development builds are replaced with the latest stable release.
func Upgrade(ctx context.Context, current string, out io.Writer) error {
	u := updater{
		client:    &http.Client{Timeout: 2 * time.Minute},
		latestURL: latestURL,
		os:        runtime.GOOS, arch: runtime.GOARCH,
		executable: os.Executable,
	}
	return u.upgrade(ctx, current, out)
}

func normalizeVersion(version string) string {
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	return version
}

func (u updater) upgrade(ctx context.Context, current string, out io.Writer) error {
	if current != "dev" && !semver.IsValid(normalizeVersion(current)) {
		return fmt.Errorf("cannot compare local version %q; reinstall qrr using the installer", current)
	}
	if _, err := fmt.Fprintln(out, "Checking for updates..."); err != nil {
		return err
	}
	metadata, err := u.fetch(ctx, u.latestURL, maxMetadataSize)
	if err != nil {
		return fmt.Errorf("check latest release: %w", err)
	}
	var latest release
	if err := json.Unmarshal(metadata, &latest); err != nil {
		return fmt.Errorf("decode latest release: %w", err)
	}
	latestVersion := normalizeVersion(latest.Tag)
	if latest.Draft || latest.Prerelease || !semver.IsValid(latestVersion) || semver.Prerelease(latestVersion) != "" {
		return fmt.Errorf("GitHub did not return a stable release with a valid semantic version")
	}
	if current != "dev" && semver.Compare(latestVersion, normalizeVersion(current)) <= 0 {
		_, err := fmt.Fprintf(out, "qrr %s is up to date (latest: %s).\n", current, latest.Tag)
		return err
	}
	archive, binary, err := archiveName(latest.Tag, u.os, u.arch)
	if err != nil {
		return err
	}
	archiveURL, err := latest.assetURL(archive)
	if err != nil {
		return err
	}
	checksumURL, err := latest.assetURL("checksums.txt")
	if err != nil {
		return err
	}
	checksums, err := u.fetch(ctx, checksumURL, maxMetadataSize)
	if err != nil {
		return fmt.Errorf("download checksums: %w", err)
	}
	expected, err := archiveChecksum(checksums, archive)
	if err != nil {
		return err
	}
	path, err := u.executable()
	if err != nil {
		return fmt.Errorf("locate executable: %w", err)
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("executable is not a regular file: %s", path)
	}
	// Staging beside the executable keeps the final rename on the same filesystem.
	dir, err := os.MkdirTemp(filepath.Dir(path), ".qrr-upgrade-")
	if err != nil {
		return fmt.Errorf("create upgrade directory beside %s (write permission required): %w", path, err)
	}
	defer os.RemoveAll(dir)
	if _, err := fmt.Fprintf(out, "Downloading qrr %s for %s/%s...\n", latest.Tag, u.os, u.arch); err != nil {
		return err
	}
	archivePath := filepath.Join(dir, archive)
	if err := u.downloadArchive(ctx, archiveURL, archivePath, expected); err != nil {
		return err
	}
	staged := filepath.Join(dir, binary)
	if err := extractBinary(ctx, archivePath, staged, binary, u.os == "windows"); err != nil {
		return fmt.Errorf("extract executable: %w", err)
	}
	if err := os.Chmod(staged, info.Mode().Perm()); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	backup, err := replaceExecutable(staged, path)
	if err != nil {
		return fmt.Errorf("replace executable %s: %w", path, err)
	}
	if _, err := fmt.Fprintf(out, "Upgraded qrr from %s to %s.\n", current, latest.Tag); err != nil {
		return err
	}
	if backup != "" {
		_, err = fmt.Fprintf(out, "Previous executable retained at %s; it can be removed after this process exits.\n", backup)
	}
	return err
}

func archiveName(tag, platform, arch string) (string, string, error) {
	if (platform != "darwin" && platform != "linux" && platform != "windows") ||
		(arch != "amd64" && arch != "arm64") || (platform == "windows" && arch != "amd64") {
		return "", "", fmt.Errorf("no release binary is available for %s/%s", platform, arch)
	}
	version := strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._-", r) {
			return r
		}
		return '_'
	}, tag)
	name := fmt.Sprintf("qrr_%s_%s_%s", version, platform, arch)
	if platform == "windows" {
		return name + ".zip", "qrr.exe", nil
	}
	return name + ".tar.gz", "qrr", nil
}

func (r release) assetURL(name string) (string, error) {
	var found string
	count := 0
	for _, asset := range r.Assets {
		if asset.Name == name {
			found = asset.URL
			count++
		}
	}
	if count != 1 || found == "" {
		return "", fmt.Errorf("release %s must contain exactly one %s asset", r.Tag, name)
	}
	return found, nil
}

func archiveChecksum(data []byte, name string) (string, error) {
	var checksum string
	count := 0
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || strings.TrimPrefix(strings.TrimPrefix(fields[1], "*"), "./") != name {
			continue
		}
		checksum = strings.ToLower(fields[0])
		count++
	}
	decoded, err := hex.DecodeString(checksum)
	if count != 1 || err != nil || len(decoded) != sha256.Size {
		return "", fmt.Errorf("missing, invalid, or duplicate checksum for %s", name)
	}
	return checksum, nil
}

func (u updater) request(ctx context.Context, address string) (*http.Response, error) {
	parsed, err := url.Parse(address)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return nil, fmt.Errorf("release URLs must use HTTPS")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "qrr-upgrader")
	if address == u.latestURL {
		req.Header.Set("Accept", "application/vnd.github+json")
	}
	client := *u.client
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.URL.Scheme != "https" {
			return fmt.Errorf("release redirects must use HTTPS")
		}
		if len(via) >= 10 {
			return fmt.Errorf("too many release redirects")
		}
		if u.client.CheckRedirect != nil {
			return u.client.CheckRedirect(req, via)
		}
		return nil
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return nil, fmt.Errorf("GitHub request failed: %s", response.Status)
	}
	return response, nil
}

func (u updater) fetch(ctx context.Context, address string, limit int64) ([]byte, error) {
	response, err := u.request(ctx, address)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("release response exceeds %d bytes", limit)
	}
	return data, nil
}

func (u updater) downloadArchive(ctx context.Context, address, path, expected string) error {
	response, err := u.request(ctx, address)
	if err != nil {
		return fmt.Errorf("download archive: %w", err)
	}
	defer response.Body.Close()
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	hash := sha256.New()
	n, err := io.Copy(io.MultiWriter(file, hash), io.LimitReader(response.Body, maxArchiveSize+1))
	if err != nil {
		return fmt.Errorf("download archive: %w", err)
	}
	if n > maxArchiveSize {
		return fmt.Errorf("release archive exceeds %d bytes", maxArchiveSize)
	}
	if hex.EncodeToString(hash.Sum(nil)) != expected {
		return fmt.Errorf("downloaded archive failed SHA-256 verification")
	}
	return file.Close()
}
