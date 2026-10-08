#!/bin/sh
# Offline integration tests for platform selection, upgrades, and failed downloads.
set -eu

repository_dir=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
workspace=$(mktemp -d)
trap 'rm -rf "$workspace"' 0
mkdir -p "$workspace/mocks" "$workspace/assets" "$workspace/source"
printf '#!/bin/sh\nprintf "qrr installer fixture\\n"\n' > "$workspace/source/qrr"
tar -czf "$workspace/archive.tar.gz" -C "$workspace/source" qrr
if command -v sha256sum >/dev/null 2>&1; then
    checksum=$(sha256sum < "$workspace/archive.tar.gz" | awk '{print $1}')
else
    checksum=$(shasum -a 256 < "$workspace/archive.tar.gz" | awk '{print $1}')
fi
for os in darwin linux; do
    for arch in amd64 arm64; do
        name="qrr_v1.2.3_${os}_${arch}.tar.gz"
        cp "$workspace/archive.tar.gz" "$workspace/assets/$name"
        printf '%s  ./%s\n' "$checksum" "$name" >> "$workspace/assets/checksums.txt"
    done
done
cp "$workspace/assets/checksums.txt" "$workspace/valid-checksums.txt"

cat > "$workspace/mocks/uname" <<'MOCK'
#!/bin/sh
case "$1" in
    -s) printf '%s\n' "$TEST_OS" ;;
    -m) printf '%s\n' "$TEST_ARCH" ;;
    *) exit 1 ;;
esac
MOCK
cat > "$workspace/mocks/curl" <<'MOCK'
#!/bin/sh
set -eu
destination=
url=
while [ "$#" -gt 0 ]; do
    case "$1" in
        -o) destination=$2; shift 2 ;;
        --proto|--proto-redir|--retry|--connect-timeout|--max-time|-w) shift 2 ;;
        https://*) url=$1; shift ;;
        *) shift ;;
    esac
done
printf '%s\n' "$url" >> "$TEST_WORKSPACE/requests"
case "$url" in
    https://github.com/harryxu/qrr/releases/latest)
        printf '%s' "${TEST_RELEASE_URL:-https://github.com/harryxu/qrr/releases/tag/v1.2.3}" ;;
    https://github.com/harryxu/qrr/releases/download/v1.2.3/*)
        [ "${TEST_DOWNLOAD_FAILURE:-0}" = 0 ] || exit 22
        cp "$TEST_WORKSPACE/assets/${url##*/}" "$destination" ;;
    *) printf 'Unexpected request: %s\n' "$url" >&2; exit 1 ;;
esac
MOCK
chmod +x "$workspace/mocks/uname" "$workspace/mocks/curl"
export TEST_WORKSPACE="$workspace"
export PATH="$workspace/mocks:$PATH"
export QRR_INSTALL_DIR="$workspace/install directory/bin"

run_install() {
    sh "$repository_dir/install.sh" > "$workspace/output" 2>&1
}

expect_failure() {
    if run_install; then
        printf 'Expected installation failure: %s\n' "$1" >&2
        exit 1
    fi
    grep -F "$1" "$workspace/output" >/dev/null
    cmp "$workspace/source/qrr" "$QRR_INSTALL_DIR/qrr"
}

for TEST_OS in Darwin Linux; do
    for TEST_ARCH in x86_64 amd64 aarch64 arm64; do
        export TEST_OS TEST_ARCH
        run_install
        cmp "$workspace/source/qrr" "$QRR_INSTALL_DIR/qrr"
        [ -x "$QRR_INSTALL_DIR/qrr" ]
        "$QRR_INSTALL_DIR/qrr" | grep -F 'qrr installer fixture' >/dev/null
    done
done
printf 'PASS: macOS/Linux architecture aliases, paths with spaces, and repeated installs\n'

export TEST_OS=FreeBSD TEST_ARCH=x86_64
expect_failure 'Supported systems are macOS and Linux'
export TEST_OS=Linux TEST_ARCH=riscv64
expect_failure 'Supported architectures are amd64 and arm64'
export TEST_ARCH=x86_64 TEST_DOWNLOAD_FAILURE=1
expect_failure 'Could not download'
unset TEST_DOWNLOAD_FAILURE
export TEST_RELEASE_URL=https://github.com/harryxu/qrr/releases
expect_failure 'GitHub did not return a published release'
unset TEST_RELEASE_URL
printf 'PASS: unsupported platforms, download failures, and missing releases\n'

printf '%064d  ./qrr_v1.2.3_linux_amd64.tar.gz\n' 0 > "$workspace/assets/checksums.txt"
expect_failure 'failed SHA-256 verification'
printf 'invalid  ./qrr_v1.2.3_linux_amd64.tar.gz\n' > "$workspace/assets/checksums.txt"
expect_failure 'Invalid checksum'
: > "$workspace/assets/checksums.txt"
expect_failure 'Missing or duplicate checksum'
cat "$workspace/valid-checksums.txt" "$workspace/valid-checksums.txt" > "$workspace/assets/checksums.txt"
expect_failure 'Missing or duplicate checksum'
printf 'PASS: invalid, missing, duplicate, and mismatched checksums preserve the installed binary\n'
