#!/usr/bin/env bash
set -euo pipefail

version=${1:?Usage: bash scripts/build-deb.sh VERSION ARCH [OUTPUT_DIR]}
arch=${2:?Usage: bash scripts/build-deb.sh VERSION ARCH [OUTPUT_DIR]}
output_dir=${3:-dist}

if [[ ! "$version" =~ ^[0-9]+(\.[0-9]+)*(-[a-zA-Z0-9.+~]+)?([+~][a-zA-Z0-9.]+)?$ ]]; then
  printf 'Invalid Debian package version: %s\n' "$version" >&2
  exit 1
fi
case "$arch" in
  amd64|arm64) ;;
  *) printf 'Unsupported architecture: %s\n' "$arch" >&2; exit 1 ;;
esac

staging=$(mktemp -d)
trap 'rm -rf "$staging"' EXIT
package_root="$staging/cu-go"
mkdir -p "$package_root/DEBIAN" "$package_root/usr/bin" "$package_root/usr/share/doc/cu-go" "$output_dir"

CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go build -trimpath -ldflags='-s -w' -o "$package_root/usr/bin/cu" .
cp README.md "$package_root/usr/share/doc/cu-go/README.md"
cp LICENSE "$package_root/usr/share/doc/cu-go/copyright"

printf 'Package: cu-go\nVersion: %s\nSection: comm\nPriority: optional\nArchitecture: %s\nMaintainer: cu-go contributors <cu-go@localhost>\nConflicts: cu\nReplaces: cu\nProvides: cu\nDescription: direct serial terminal client written in Go\n Connect to a serial device and exchange data interactively.\n Supports the cu direct connection mode and basic escape commands.\n' "$version" "$arch" > "$package_root/DEBIAN/control"

dpkg-deb --build --root-owner-group "$package_root" "$output_dir/cu-go_${version}_${arch}.deb"
