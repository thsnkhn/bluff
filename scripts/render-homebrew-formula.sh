#!/usr/bin/env bash
set -euo pipefail

version_tag="${1:?usage: render-homebrew-formula.sh VERSION_TAG CHECKSUMS_FILE [OUTPUT_FILE]}"
checksums_file="${2:?usage: render-homebrew-formula.sh VERSION_TAG CHECKSUMS_FILE [OUTPUT_FILE]}"
output_file="${3:-Formula/bluff.rb}"
repository="${GITHUB_REPOSITORY:-thsnkhn/bluff}"
version="${version_tag#v}"

checksum_for() {
  local filename="$1"
  local checksum
  checksum="$(awk -v filename="$filename" '$2 == filename || $2 == "./" filename { print $1 }' "$checksums_file")"
  if [[ ! "$checksum" =~ ^[0-9a-f]{64}$ ]]; then
    echo "missing or invalid checksum for ${filename}" >&2
    exit 1
  fi
  printf '%s' "$checksum"
}

darwin_arm64="bluff_${version_tag}_darwin_arm64.tar.gz"
darwin_amd64="bluff_${version_tag}_darwin_amd64.tar.gz"
linux_arm64="bluff_${version_tag}_linux_arm64.tar.gz"
linux_amd64="bluff_${version_tag}_linux_amd64.tar.gz"

mkdir -p "$(dirname "$output_file")"
cat > "$output_file" <<RUBY
class Bluff < Formula
  desc "Private poker ledger for the terminal"
  homepage "https://github.com/${repository}"
  version "${version}"
  license "GPL-3.0-only"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/${repository}/releases/download/${version_tag}/${darwin_arm64}"
      sha256 "$(checksum_for "$darwin_arm64")"
    else
      url "https://github.com/${repository}/releases/download/${version_tag}/${darwin_amd64}"
      sha256 "$(checksum_for "$darwin_amd64")"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/${repository}/releases/download/${version_tag}/${linux_arm64}"
      sha256 "$(checksum_for "$linux_arm64")"
    else
      url "https://github.com/${repository}/releases/download/${version_tag}/${linux_amd64}"
      sha256 "$(checksum_for "$linux_amd64")"
    end
  end

  def install
    bin.install "bluff"
  end

  test do
    assert_match "bluff v#{version}", shell_output("#{bin}/bluff --version")
  end
end
RUBY
