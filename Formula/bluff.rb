class Bluff < Formula
  desc "Private poker ledger for the terminal"
  homepage "https://github.com/thsnkhn/bluff"
  version "0.1.7"
  license "GPL-3.0-only"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.7/bluff_v0.1.7_darwin_arm64.tar.gz"
      sha256 "62941fac7b4528288aa054276a6e387be4171c032ca7a64d49017cce37aeb8d0"
    else
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.7/bluff_v0.1.7_darwin_amd64.tar.gz"
      sha256 "4f27f0d98d39e4dda5efd8b91abe307566e66087b75f80c88c9cdf37c8ec470f"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.7/bluff_v0.1.7_linux_arm64.tar.gz"
      sha256 "f3c63549a6754f6a5708f4c93dece9bd77661e956cc3c69559996812ccf99f7d"
    else
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.7/bluff_v0.1.7_linux_amd64.tar.gz"
      sha256 "c3028a215cd4f265f8bf3bfba6d7788e16c5555055487c1f3d57176a9c994938"
    end
  end

  def install
    bin.install "bluff"
  end

  test do
    assert_match "bluff v#{version}", shell_output("#{bin}/bluff --version")
  end
end
