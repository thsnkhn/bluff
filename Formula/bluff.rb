class Bluff < Formula
  desc "Private poker ledger for the terminal"
  homepage "https://github.com/thsnkhn/bluff"
  version "0.1.0"
  license "GPL-3.0-only"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.0/bluff_v0.1.0_darwin_arm64.tar.gz"
      sha256 "dc3b41e2340c8642c4311b3fb5f3097e625168bd3ffe135c093762cc8cb8ed98"
    else
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.0/bluff_v0.1.0_darwin_amd64.tar.gz"
      sha256 "4bd34913e160ee47516cc4930679d71da74a5aad6bf9be8f9258f54bc92fa17f"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.0/bluff_v0.1.0_linux_arm64.tar.gz"
      sha256 "7d8390743118f7476075d2f3b25cff0084c35412460f3307f0ee12750e7c0a32"
    else
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.0/bluff_v0.1.0_linux_amd64.tar.gz"
      sha256 "24938b4cd2c4867e9d9a5fc6f6268476ad54162c4cbf92434242dc64af1407e6"
    end
  end

  def install
    bin.install "bluff"
  end

  test do
    assert_match "bluff v#{version}", shell_output("#{bin}/bluff --version")
  end
end
