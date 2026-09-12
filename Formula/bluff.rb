class Bluff < Formula
  desc "Private poker ledger for the terminal"
  homepage "https://github.com/thsnkhn/bluff"
  version "0.1.11"
  license "GPL-3.0-only"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.11/bluff_v0.1.11_darwin_arm64.tar.gz"
      sha256 "94a5183c4187a28e265f68d64d1c73a4174a306d74986de4cd406a2ac9fade5f"
    else
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.11/bluff_v0.1.11_darwin_amd64.tar.gz"
      sha256 "d850696346ce2061016c838081d2e3db817f1847c82746a4a5f07f0f5410519c"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.11/bluff_v0.1.11_linux_arm64.tar.gz"
      sha256 "e135c434afbdfcf5c38a8ef2adb9dfa3fe5f1593ba009e296ddc7b40a3bc8cbc"
    else
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.11/bluff_v0.1.11_linux_amd64.tar.gz"
      sha256 "1e263152b45d9763b1ab615db449cf7e323ee5094b79bdfab2fda4702eb6ab2f"
    end
  end

  def install
    bin.install "bluff"
  end

  test do
    assert_match "bluff v#{version}", shell_output("#{bin}/bluff --version")
  end
end
