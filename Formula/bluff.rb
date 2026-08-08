class Bluff < Formula
  desc "Private poker ledger for the terminal"
  homepage "https://github.com/thsnkhn/bluff"
  version "0.1.5"
  license "GPL-3.0-only"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.5/bluff_v0.1.5_darwin_arm64.tar.gz"
      sha256 "0a5886cb51fb4ab6e70dae13bc23b001d5de12c2e1b8db7e56f738067f96e5cd"
    else
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.5/bluff_v0.1.5_darwin_amd64.tar.gz"
      sha256 "b222d2d0536ff204b4dcae49f1cbdf7a579f69feabd47e8960a215a7d1a5bd9c"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.5/bluff_v0.1.5_linux_arm64.tar.gz"
      sha256 "8c6f2b4b38001c965444c6949ceb1a465de5b00c34eaa8ed2f997ce92f958e31"
    else
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.5/bluff_v0.1.5_linux_amd64.tar.gz"
      sha256 "b026600751707d67ddf350f4893c763c1aae3742c35df3bc022cff79b44b4669"
    end
  end

  def install
    bin.install "bluff"
  end

  test do
    assert_match "bluff v#{version}", shell_output("#{bin}/bluff --version")
  end
end
