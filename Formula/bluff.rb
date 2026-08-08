class Bluff < Formula
  desc "Private poker ledger for the terminal"
  homepage "https://github.com/thsnkhn/bluff"
  version "0.1.6"
  license "GPL-3.0-only"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.6/bluff_v0.1.6_darwin_arm64.tar.gz"
      sha256 "07d110ef7f56b35d30f6d8a12993f6bae9725820e393493e92f6fc076d02f6ed"
    else
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.6/bluff_v0.1.6_darwin_amd64.tar.gz"
      sha256 "472de6a2b8670f4e096c7a74f95a619e1161c9dd93eeac143027fb6294c6f9d8"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.6/bluff_v0.1.6_linux_arm64.tar.gz"
      sha256 "541d08286fa87e7b9b5ab31950173ef20c061b2b7c3d3f1ff07895add72e9d7d"
    else
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.6/bluff_v0.1.6_linux_amd64.tar.gz"
      sha256 "97e8c8789388a7cd6a27854ace2f7d2f41a58b59c9cc9baed5cde61656f054c3"
    end
  end

  def install
    bin.install "bluff"
  end

  test do
    assert_match "bluff v#{version}", shell_output("#{bin}/bluff --version")
  end
end
