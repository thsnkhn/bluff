class Bluff < Formula
  desc "Private poker ledger for the terminal"
  homepage "https://github.com/thsnkhn/bluff"
  version "0.1.4"
  license "GPL-3.0-only"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.4/bluff_v0.1.4_darwin_arm64.tar.gz"
      sha256 "3bdce58ffa0feff651d43ccdf6b41c5ecf24281e18cf5f7ba9223b864ee8b801"
    else
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.4/bluff_v0.1.4_darwin_amd64.tar.gz"
      sha256 "1d581aedc15b329b1a68fed683922e3991eafbe1f5324dbc37f4d4ce0c091fca"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.4/bluff_v0.1.4_linux_arm64.tar.gz"
      sha256 "bfdca3a938cb2cd780f4ebc1e8e36dfb7bf08f41247b59f6ecbc9d95c55af583"
    else
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.4/bluff_v0.1.4_linux_amd64.tar.gz"
      sha256 "8a1b52e293e83e53040bf9f103d34515efc98ad9ffbbb51dba63c92df566b9f0"
    end
  end

  def install
    bin.install "bluff"
  end

  test do
    assert_match "bluff v#{version}", shell_output("#{bin}/bluff --version")
  end
end
