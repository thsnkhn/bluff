class Bluff < Formula
  desc "Private poker ledger for the terminal"
  homepage "https://github.com/thsnkhn/bluff"
  version "0.1.12"
  license "GPL-3.0-only"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.12/bluff_v0.1.12_darwin_arm64.tar.gz"
      sha256 "9078385f8bc61a1d6120fa515af02dd4ea3d38f998c40b90e4bdffc7b8623b90"
    else
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.12/bluff_v0.1.12_darwin_amd64.tar.gz"
      sha256 "2236813c7d49e7a0c0ba3c87b314bf986b20e00b467ddc91d73d0633068ed85e"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.12/bluff_v0.1.12_linux_arm64.tar.gz"
      sha256 "5a9aee085c81e5f2bbb017cb04e05da36b5999a4f4beaac14556e3eeb1ee4406"
    else
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.12/bluff_v0.1.12_linux_amd64.tar.gz"
      sha256 "ab487774a4c4ccb3afcb1c452d8b213524213ad0fe9c223c419a436c3586b8d6"
    end
  end

  def install
    bin.install "bluff"
  end

  test do
    assert_match "bluff v#{version}", shell_output("#{bin}/bluff --version")
  end
end
