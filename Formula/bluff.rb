class Bluff < Formula
  desc "Private poker ledger for the terminal"
  homepage "https://github.com/thsnkhn/bluff"
  version "0.1.3"
  license "GPL-3.0-only"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.3/bluff_v0.1.3_darwin_arm64.tar.gz"
      sha256 "6572dccd3304efd257785d5a077999f1261fef5df22d950f4e1999ae1379495b"
    else
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.3/bluff_v0.1.3_darwin_amd64.tar.gz"
      sha256 "7927f996caeb6efdf8cbba9c9f3f7538d0f8cfb83d3d65432aae7c9f3d012668"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.3/bluff_v0.1.3_linux_arm64.tar.gz"
      sha256 "b6ec3d0bf8ce65c63008a068ba1438b9904116d007814599c45a7b1641ec5737"
    else
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.3/bluff_v0.1.3_linux_amd64.tar.gz"
      sha256 "205bbaca24f93a4b6a4e025fb1a017ab67e7028801750838c93516c0a7937347"
    end
  end

  def install
    bin.install "bluff"
  end

  test do
    assert_match "bluff v#{version}", shell_output("#{bin}/bluff --version")
  end
end
