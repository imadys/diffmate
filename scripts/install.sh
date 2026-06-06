#!/usr/bin/env sh
set -eu

bin_name="diffmate"
install_dir="${DIFFMATE_INSTALL_DIR:-/usr/local/bin}"

if [ ! -d "$install_dir" ]; then
  mkdir -p "$install_dir"
fi

go build -o "$install_dir/$bin_name" ./cmd/diffmate

echo "Installed $bin_name to $install_dir/$bin_name"
echo "Run: $bin_name review"
