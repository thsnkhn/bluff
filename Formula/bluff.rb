class Bluff < Formula
  desc "Private poker ledger for the terminal"
  homepage "https://github.com/thsnkhn/bluff"
  version "0.1.10"
  license "GPL-3.0-only"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.10/bluff_v0.1.10_darwin_arm64.tar.gz"
      sha256 "c76f88cd801a5138dcde9247b785958d982ce9caa3452eee53afe1973e3a649b"
    else
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.10/bluff_v0.1.10_darwin_amd64.tar.gz"
      sha256 "65b33906f7ca145e12ce50dd98a407368c2ab9d4acce1ef43bd46d09b790bd73"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.10/bluff_v0.1.10_linux_arm64.tar.gz"
      sha256 "7471946550289d2ea00eae25407a18610ea9bc3fe4c4813a90dc3a2e338f7234"
    else
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.10/bluff_v0.1.10_linux_amd64.tar.gz"
      sha256 "ff06a7b7f5ab4ceb10f75241ff22d53f1fc95271aa2b674ad86a5d36e73f3316"
    end
  end

  def install
    bin.install "bluff"
  end

  test do
    assert_match "bluff v#{version}", shell_output("#{bin}/bluff --version")
  end
end
