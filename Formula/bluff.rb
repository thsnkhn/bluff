class Bluff < Formula
  desc "Private poker ledger for the terminal"
  homepage "https://github.com/thsnkhn/bluff"
  version "0.1.1"
  license "GPL-3.0-only"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.1/bluff_v0.1.1_darwin_arm64.tar.gz"
      sha256 "91ab7c8a4f265afe344add5491e6bad545951d9705aad40be0f8c35cdae9027e"
    else
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.1/bluff_v0.1.1_darwin_amd64.tar.gz"
      sha256 "80a61aff74c04bda01a3f447a340dc45b153f6ea3b63267d1c5813053f1a4832"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.1/bluff_v0.1.1_linux_arm64.tar.gz"
      sha256 "bf746c42d670c1beb45bfbd1ab94a6f8bd0ffa3fb6b3c02776140243ca51bcf4"
    else
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.1/bluff_v0.1.1_linux_amd64.tar.gz"
      sha256 "bb44c7b30e0ff10ca94fcdf2012121d0712696dee0fcfc6acb850d47fff887be"
    end
  end

  def install
    bin.install "bluff"
  end

  test do
    assert_match "bluff v#{version}", shell_output("#{bin}/bluff --version")
  end
end
