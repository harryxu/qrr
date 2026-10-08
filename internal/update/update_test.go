package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

const updatedBinary = "new qrr executable"

type fixture struct {
	u             updater
	target, name  string
	release       release
	archive       []byte
	checksums     string
	metadata      []byte
	statuses      map[string]int
	cancelArchive context.CancelFunc
	mu            sync.Mutex
	requests      []string
	certificate   []byte
}

func archiveFixture(t *testing.T, zipped bool, names ...string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	if zipped {
		writer := zip.NewWriter(&buffer)
		for _, name := range names {
			entry, err := writer.Create(name)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := io.WriteString(entry, updatedBinary); err != nil {
				t.Fatal(err)
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
	} else {
		gz := gzip.NewWriter(&buffer)
		writer := tar.NewWriter(gz)
		for _, name := range names {
			header := &tar.Header{Name: name, Mode: 0755, Size: int64(len(updatedBinary)), Typeflag: tar.TypeReg}
			if err := writer.WriteHeader(header); err != nil {
				t.Fatal(err)
			}
			if _, err := io.WriteString(writer, updatedBinary); err != nil {
				t.Fatal(err)
			}
		}
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		if err := gz.Close(); err != nil {
			t.Fatal(err)
		}
	}
	return buffer.Bytes()
}

func newFixture(t *testing.T, platform, arch string) *fixture {
	t.Helper()
	name, binary, err := archiveName("v0.10.0", platform, arch)
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{name: name, statuses: map[string]int{}}
	f.target = filepath.Join(t.TempDir(), "installed qrr"+filepath.Ext(binary))
	if err := os.WriteFile(f.target, []byte("previous executable"), 0751); err != nil {
		t.Fatal(err)
	}
	f.archive = archiveFixture(t, platform == "windows", binary)
	f.refreshChecksum()
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		f.requests = append(f.requests, r.URL.Path)
		f.mu.Unlock()
		if status := f.statuses[r.URL.Path]; status != 0 {
			w.WriteHeader(status)
			return
		}
		switch r.URL.Path {
		case "/latest":
			if len(f.metadata) > 0 {
				_, _ = w.Write(f.metadata)
				return
			}
			_ = json.NewEncoder(w).Encode(f.release)
		case "/checksums":
			_, _ = io.WriteString(w, f.checksums)
		case "/archive":
			if f.cancelArchive != nil {
				_, _ = w.Write(f.archive[:len(f.archive)/2])
				w.(http.Flusher).Flush()
				f.cancelArchive()
				<-r.Context().Done()
				return
			}
			_, _ = w.Write(f.archive)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	f.certificate = server.Certificate().Raw
	f.release = release{Tag: "v0.10.0", Assets: []asset{{Name: name, URL: server.URL + "/archive"}, {Name: "checksums.txt", URL: server.URL + "/checksums"}}}
	f.u = updater{client: server.Client(), latestURL: server.URL + "/latest", os: platform, arch: arch, executable: func() (string, error) { return f.target, nil }}
	return f
}

func (f *fixture) refreshChecksum() {
	hash := sha256.Sum256(f.archive)
	f.checksums = fmt.Sprintf("%s  ./%s\n", hex.EncodeToString(hash[:]), f.name)
}

func (f *fixture) assertOriginal(t *testing.T) {
	t.Helper()
	data, err := os.ReadFile(f.target)
	if err != nil || string(data) != "previous executable" {
		t.Fatalf("previous executable changed: %q, %v", data, err)
	}
	entries, err := os.ReadDir(filepath.Dir(f.target))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("upgrade left temporary files: %v", entries)
	}
}

func TestUpgradeInstallsVerifiedRelease(t *testing.T) {
	for _, platform := range []struct{ os, arch string }{{"darwin", "amd64"}, {"darwin", "arm64"}, {"linux", "amd64"}, {"linux", "arm64"}, {"windows", "amd64"}} {
		t.Run(platform.os+"/"+platform.arch, func(t *testing.T) {
			f := newFixture(t, platform.os, platform.arch)
			var output bytes.Buffer
			if err := f.u.upgrade(context.Background(), "v0.9.0", &output); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(f.target)
			if err != nil || string(data) != updatedBinary {
				t.Fatalf("installed executable=%q, error=%v", data, err)
			}
			info, err := os.Stat(f.target)
			if err != nil {
				t.Fatal(err)
			}
			if runtime.GOOS != "windows" && info.Mode().Perm() != 0751 {
				t.Fatalf("permissions changed: %v", info.Mode())
			}
			if !strings.Contains(output.String(), "Upgraded qrr from v0.9.0 to v0.10.0.") {
				t.Fatal(output.String())
			}
		})
	}
}

func TestUpgradeSkipsSameOrOlderReleases(t *testing.T) {
	for _, current := range []string{"v0.10.0", "0.10.0", "v0.11.0", "v1.0.0", "v0.10.0+local"} {
		t.Run(current, func(t *testing.T) {
			f := newFixture(t, "linux", "amd64")
			if err := f.u.upgrade(context.Background(), current, io.Discard); err != nil {
				t.Fatal(err)
			}
			f.assertOriginal(t)
			f.mu.Lock()
			defer f.mu.Unlock()
			if len(f.requests) != 1 || f.requests[0] != "/latest" {
				t.Fatalf("unnecessary downloads: %v", f.requests)
			}
		})
	}
}

func TestUpgradeDevelopmentAndPrereleaseBuilds(t *testing.T) {
	for _, current := range []string{"dev", "v0.10.0-rc.1"} {
		f := newFixture(t, "linux", "amd64")
		if err := f.u.upgrade(context.Background(), current, io.Discard); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(f.target)
		if err != nil || string(data) != updatedBinary {
			t.Fatalf("upgrade failed for %s: %q %v", current, data, err)
		}
	}
}

func TestUpgradeFailuresPreserveExecutable(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		configure  func(*fixture)
	}{
		{"invalid local version", "cannot compare local version", func(f *fixture) {}},
		{"invalid remote version", "stable release", func(f *fixture) { f.release.Tag = "broken" }},
		{"draft", "stable release", func(f *fixture) { f.release.Draft = true }},
		{"prerelease", "stable release", func(f *fixture) { f.release.Prerelease = true }},
		{"prerelease tag", "stable release", func(f *fixture) { f.release.Tag = "v1.0.0-rc.1" }},
		{"missing assets", "exactly one", func(f *fixture) { f.release.Assets = nil }},
		{"duplicate asset", "exactly one", func(f *fixture) { f.release.Assets = append(f.release.Assets, f.release.Assets[0]) }},
		{"missing checksum", "checksum", func(f *fixture) { f.checksums = "" }},
		{"invalid checksum", "checksum", func(f *fixture) { f.checksums = "invalid  ./" + f.name }},
		{"duplicate checksum", "checksum", func(f *fixture) { f.checksums += f.checksums }},
		{"checksum mismatch", "SHA-256", func(f *fixture) { f.archive = append(f.archive, 'x') }},
		{"API rate limit", "403", func(f *fixture) { f.statuses["/latest"] = 403 }},
		{"download failure", "404", func(f *fixture) { f.statuses["/archive"] = 404 }},
		{"checksum download failure", "500", func(f *fixture) { f.statuses["/checksums"] = 500 }},
		{"invalid JSON", "decode", func(f *fixture) { f.metadata = []byte("invalid") }},
		{"oversized metadata", "exceeds", func(f *fixture) { f.metadata = bytes.Repeat([]byte("x"), maxMetadataSize+1) }},
		{"unsupported architecture", "no release binary", func(f *fixture) { f.u.arch = "386" }},
		{"insecure asset URL", "HTTPS", func(f *fixture) { f.release.Assets[0].URL = "http://example.invalid/archive" }},
		{"broken archive", "extract", func(f *fixture) { f.archive = []byte("not an archive"); f.refreshChecksum() }},
		{"wrong executable", "archive", func(f *fixture) { f.archive = archiveFixture(t, false, "wrong"); f.refreshChecksum() }},
		{"path traversal", "archive", func(f *fixture) { f.archive = archiveFixture(t, false, "../qrr"); f.refreshChecksum() }},
		{"duplicate executable", "archive", func(f *fixture) { f.archive = archiveFixture(t, false, "qrr", "qrr"); f.refreshChecksum() }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t, "linux", "amd64")
			tc.configure(f)
			current := "v0.9.0"
			if tc.name == "invalid local version" {
				current = "custom-build"
			}
			err := f.u.upgrade(context.Background(), current, io.Discard)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
			f.assertOriginal(t)
		})
	}
}

func TestUpgradeCancellationPreservesExecutable(t *testing.T) {
	for _, duringDownload := range []bool{false, true} {
		f := newFixture(t, "linux", "amd64")
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if duringDownload {
			f.cancelArchive = cancel
		} else {
			cancel()
		}
		err := f.u.upgrade(ctx, "v0.9.0", io.Discard)
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected cancellation, got %v", err)
		}
		f.assertOriginal(t)
	}
}

func TestUpgradeRejectsLinkedAndCorruptExecutables(t *testing.T) {
	for _, platform := range []string{"linux", "windows"} {
		for _, corrupt := range []bool{false, true} {
			f := newFixture(t, platform, "amd64")
			if corrupt {
				if platform == "linux" {
					f.archive[len(f.archive)-8] ^= 1 // Corrupt the gzip CRC, with a matching outer SHA-256.
				} else {
					f.archive[30+len("qrr.exe")] ^= 0xff // Corrupt the ZIP's compressed payload.
				}
			} else {
				var buffer bytes.Buffer
				if platform == "linux" {
					gz := gzip.NewWriter(&buffer)
					writer := tar.NewWriter(gz)
					if err := writer.WriteHeader(&tar.Header{Name: "qrr", Typeflag: tar.TypeSymlink, Linkname: "../outside"}); err != nil {
						t.Fatal(err)
					}
					if err := writer.Close(); err != nil {
						t.Fatal(err)
					}
					if err := gz.Close(); err != nil {
						t.Fatal(err)
					}
				} else {
					writer := zip.NewWriter(&buffer)
					header := &zip.FileHeader{Name: "qrr.exe"}
					header.SetMode(os.ModeSymlink | 0777)
					entry, err := writer.CreateHeader(header)
					if err != nil {
						t.Fatal(err)
					}
					if _, err := io.WriteString(entry, "../outside"); err != nil {
						t.Fatal(err)
					}
					if err := writer.Close(); err != nil {
						t.Fatal(err)
					}
				}
				f.archive = buffer.Bytes()
			}
			f.refreshChecksum()
			if err := f.u.upgrade(context.Background(), "v0.9.0", io.Discard); err == nil {
				t.Fatalf("accepted linked/corrupt %s executable", platform)
			}
			f.assertOriginal(t)
		}
	}
}

func TestUpgradeRequiresWriteAccess(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("Unix directory permissions required")
	}
	f := newFixture(t, "linux", "amd64")
	dir := filepath.Dir(f.target)
	if err := os.Chmod(dir, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0755) })
	err := f.u.upgrade(context.Background(), "v0.9.0", io.Discard)
	if err == nil || !strings.Contains(err.Error(), "write permission required") {
		t.Fatalf("expected permission error, got %v", err)
	}
	f.assertOriginal(t)
}

func TestUpgradeSymlinkTargetsExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires Windows privileges")
	}
	f := newFixture(t, "linux", "amd64")
	link := filepath.Join(t.TempDir(), "qrr")
	if err := os.Symlink(f.target, link); err != nil {
		t.Fatal(err)
	}
	f.u.executable = func() (string, error) { return link, nil }
	if err := f.u.upgrade(context.Background(), "v0.9.0", io.Discard); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(link)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("symlink replaced: %v %v", info, err)
	}
	data, err := os.ReadFile(link)
	if err != nil || string(data) != updatedBinary {
		t.Fatalf("symlink target not upgraded: %q %v", data, err)
	}
}

func TestFailedReplacementPreservesExecutable(t *testing.T) {
	f := newFixture(t, "linux", "amd64")
	if _, err := replaceExecutable(filepath.Join(filepath.Dir(f.target), "missing"), f.target); err == nil {
		t.Fatal("missing staged executable accepted")
	}
	f.assertOriginal(t)
}

func TestUpgradeRunningExecutableChild(t *testing.T) {
	address := os.Getenv("QRR_UPGRADE_TEST_API")
	if address == "" {
		return
	}
	// Trust the loopback fixture's certificate without disabling TLS verification.
	der, err := base64.StdEncoding.DecodeString(os.Getenv("QRR_UPGRADE_TEST_CERT"))
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(certificate)
	client := &http.Client{Timeout: 10 * time.Second, Transport: &http.Transport{TLSClientConfig: &tls.Config{RootCAs: roots}}}
	u := updater{client: client, latestURL: address, os: runtime.GOOS, arch: runtime.GOARCH, executable: os.Executable}
	if err := u.upgrade(context.Background(), "v0.9.0", os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(0)
}

func TestUpgradeRunningExecutable(t *testing.T) {
	f := newFixture(t, runtime.GOOS, runtime.GOARCH)
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(self)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f.target, data, 0755); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, f.target, "-test.run=^TestUpgradeRunningExecutableChild$")
	cmd.Env = append(os.Environ(), "QRR_UPGRADE_TEST_API="+f.u.latestURL, "QRR_UPGRADE_TEST_CERT="+base64.StdEncoding.EncodeToString(f.certificate))
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("upgrade running executable: %v\n%s", err, output)
	}
	data, err = os.ReadFile(f.target)
	if err != nil || string(data) != updatedBinary {
		t.Fatalf("running executable not replaced: %q %v", data, err)
	}
}
