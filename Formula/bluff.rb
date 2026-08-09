class Bluff < Formula
  desc "Private poker ledger for the terminal"
  homepage "https://github.com/thsnkhn/bluff"
  version "0.1.9"
  license "GPL-3.0-only"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.9/bluff_v0.1.9_darwin_arm64.tar.gz"
      sha256 "028f9ad0fe32c2946b62ab7fa4975c4f509147783999423ad31a933714d2b3b9"
    else
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.9/bluff_v0.1.9_darwin_amd64.tar.gz"
      sha256 "947e0e51df16a9bfbcec4032b505fc6eac2e20d6026ae9dd41cfd1ee24eb4561"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.9/bluff_v0.1.9_linux_arm64.tar.gz"
      sha256 "845ae5fe42c069c28e4202c2f52afab02382f4da880eec79fd4792aab050eacf"
    else
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.9/bluff_v0.1.9_linux_amd64.tar.gz"
      sha256 "c574f16e892caec0a47eebf2cebbd50dc36e2a0c134b51d4349542ab9396ec88"
    end
  end

  def install
    bin.install "bluff"
  end

  test do
    assert_match "bluff v#{version}", shell_output("#{bin}/bluff --version")
  end
end
