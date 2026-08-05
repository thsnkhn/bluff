class Bluff < Formula
  desc "Private poker ledger for the terminal"
  homepage "https://github.com/thsnkhn/bluff"
  version "0.1.2"
  license "GPL-3.0-only"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.2/bluff_v0.1.2_darwin_arm64.tar.gz"
      sha256 "b055fa6bfaac3301e3befa7def56066adfbcf048f952b1ebaf409664fcb70b47"
    else
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.2/bluff_v0.1.2_darwin_amd64.tar.gz"
      sha256 "f2924719d035d3fe54680eda07da61ca7c57101258642bc53d954550ec331186"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.2/bluff_v0.1.2_linux_arm64.tar.gz"
      sha256 "27bf9b7e104c4b16a4a81611c8d9ca4f30ea55a551cf77da1bf8a8af283a7193"
    else
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.2/bluff_v0.1.2_linux_amd64.tar.gz"
      sha256 "a60b7bd34a4d300d6e5d2d9c8d40bcf76ff6c0684882de5a1b8ad0556506cd2d"
    end
  end

  def install
    bin.install "bluff"
  end

  test do
    assert_match "bluff v#{version}", shell_output("#{bin}/bluff --version")
  end
end
