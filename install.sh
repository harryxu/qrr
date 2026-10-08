#!/bin/sh
set -eu

repository=https://github.com/harryxu/qrr
install_dir=${QRR_INSTALL_DIR:-"$HOME/.local/bin"}
temporary_dir=
staged_binary=

fail() {
    printf 'Error: %s\n' "$1" >&2
    exit 1
}

cleanup() {
    if [ -n "$staged_binary" ]; then
        rm -f "$staged_binary"
    fi
    if [ -n "$temporary_dir" ]; then
        rm -rf "$temporary_dir"
    fi
}

download() {
    curl -fsSL --proto '=https' --proto-redir '=https' --retry 3 \
        --connect-timeout 10 --max-time 120 "$1" -o "$2"
}

main() {
    case "$(uname -s)" in
        Darwin) os=darwin ;;
        Linux) os=linux ;;
        *) fail 'Supported systems are macOS and Linux. Use install.ps1 on Windows.' ;;
    esac
    case "$(uname -m)" in
        x86_64|amd64) arch=amd64 ;;
        aarch64|arm64) arch=arm64 ;;
        *) fail 'Supported architectures are amd64 and arm64.' ;;
    esac

    for dependency in curl tar awk mktemp cp chmod mkdir mv rm; do
        command -v "$dependency" >/dev/null 2>&1 || fail "Required command not found: $dependency"
    done
    if command -v sha256sum >/dev/null 2>&1; then
        checksum_tool=sha256sum
    elif command -v shasum >/dev/null 2>&1; then
        checksum_tool=shasum
    elif command -v openssl >/dev/null 2>&1; then
        checksum_tool=openssl
    else
        fail 'SHA-256 verification requires sha256sum, shasum, or openssl.'
    fi

    printf 'Detecting the latest qrr release for %s/%s...\n' "$os" "$arch"
    # Resolve once so the archive and checksum always come from the same release.
    release_url=$(curl -fsSLI --proto '=https' --proto-redir '=https' --retry 3 \
        --connect-timeout 10 --max-time 30 -o /dev/null -w '%{url_effective}' \
        "$repository/releases/latest") || fail 'Could not resolve the latest release.'
    case "$release_url" in
        "$repository/releases/tag/"*) tag=${release_url#"$repository/releases/tag/"} ;;
        *) fail 'GitHub did not return a published release.' ;;
    esac
    case "$tag" in
        ''|*[!a-zA-Z0-9._-]*) fail "Unsupported release tag: $tag" ;;
    esac

    archive="qrr_${tag}_${os}_${arch}.tar.gz"
    base_url="$repository/releases/download/$tag"
    temporary_dir=$(mktemp -d)
    trap cleanup 0
    trap 'exit 130' INT
    trap 'exit 143' TERM
    printf 'Downloading qrr %s...\n' "$tag"
    download "$base_url/$archive" "$temporary_dir/$archive" || fail "Could not download $archive."
    download "$base_url/checksums.txt" "$temporary_dir/checksums.txt" || fail 'Could not download checksums.txt.'

    expected=$(awk -v file="$archive" '
        $2 == file || $2 == "./" file { print tolower($1); matches++ }
        END { if (matches != 1) exit 1 }
    ' "$temporary_dir/checksums.txt") || fail "Missing or duplicate checksum for $archive."
    case "$expected" in
        ''|*[!0-9a-f]*) fail "Invalid checksum for $archive." ;;
    esac
    [ "${#expected}" -eq 64 ] || fail "Invalid checksum for $archive."
    case "$checksum_tool" in
        sha256sum) actual=$(sha256sum < "$temporary_dir/$archive" | awk '{ print $1 }') ;;
        shasum) actual=$(shasum -a 256 < "$temporary_dir/$archive" | awk '{ print $1 }') ;;
        openssl) actual=$(openssl dgst -sha256 < "$temporary_dir/$archive" | awk '{ print $NF }') ;;
    esac
    [ "$actual" = "$expected" ] || fail 'Downloaded archive failed SHA-256 verification.'

    tar -xzf "$temporary_dir/$archive" -C "$temporary_dir" qrr || fail 'Could not extract qrr.'
    [ -f "$temporary_dir/qrr" ] && [ ! -L "$temporary_dir/qrr" ] || fail 'Archive does not contain a regular qrr executable.'
    mkdir -p "$install_dir"
    # Stage in the destination directory before replacing an existing executable.
    staged_binary=$(mktemp "$install_dir/.qrr.XXXXXX")
    cp "$temporary_dir/qrr" "$staged_binary"
    chmod 755 "$staged_binary"
    mv -f "$staged_binary" "$install_dir/qrr"
    staged_binary=

    printf 'Installed qrr %s to %s/qrr\n' "$tag" "$install_dir"
    case ":${PATH}:" in
        *":$install_dir:"*) printf 'Run qrr --help to get started.\n' ;;
        *)
            printf 'Add %s to PATH in your shell profile, then open a new terminal.\n' "$install_dir"
            if [ "$install_dir" = "$HOME/.local/bin" ]; then
                printf '  export PATH="$HOME/.local/bin:$PATH"\n'
            fi
            ;;
    esac
}

main "$@"
