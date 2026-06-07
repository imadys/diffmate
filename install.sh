#!/bin/sh
set -eu

repo="${DIFFMATE_REPO:-imadys/diffmate}"
bin_name="diffmate"
install_dir="${DIFFMATE_INSTALL_DIR:-/usr/local/bin}"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"

case "$os" in
  linux) os="linux" ;;
  darwin) os="darwin" ;;
  *)
    echo "diffmate: unsupported OS: $os" >&2
    exit 1
    ;;
esac

case "$arch" in
  x86_64 | amd64) arch="amd64" ;;
  arm64 | aarch64) arch="arm64" ;;
  *)
    echo "diffmate: unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

if ! command -v curl >/dev/null 2>&1; then
  echo "diffmate: curl is required to install" >&2
  exit 1
fi

asset="diffmate-${os}-${arch}.tar.gz"
url="https://github.com/${repo}/releases/latest/download/${asset}"
tmp_dir="$(mktemp -d)"

cleanup() {
  rm -rf "$tmp_dir"
}
trap cleanup EXIT INT TERM

echo "Downloading ${url}"
curl -fsSL "$url" -o "$tmp_dir/$asset"
tar -xzf "$tmp_dir/$asset" -C "$tmp_dir"

if [ ! -f "$tmp_dir/$bin_name" ]; then
  echo "diffmate: release archive did not contain $bin_name" >&2
  exit 1
fi

if [ -d "$install_dir" ] && [ -w "$install_dir" ]; then
  install -m 755 "$tmp_dir/$bin_name" "$install_dir/$bin_name"
else
  if command -v sudo >/dev/null 2>&1; then
    sudo mkdir -p "$install_dir"
    sudo install -m 755 "$tmp_dir/$bin_name" "$install_dir/$bin_name"
  else
    install_dir="$HOME/.local/bin"
    mkdir -p "$install_dir"
    install -m 755 "$tmp_dir/$bin_name" "$install_dir/$bin_name"
  fi
fi

echo "Installed diffmate to $install_dir/$bin_name"

case ":$PATH:" in
  *":$install_dir:"*) ;;
  *)
    echo "Add $install_dir to your PATH to run diffmate from anywhere."
    ;;
esac
