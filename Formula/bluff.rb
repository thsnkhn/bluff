class Bluff < Formula
  desc "Private poker ledger for the terminal"
  homepage "https://github.com/thsnkhn/bluff"
  version "0.1.8"
  license "GPL-3.0-only"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.8/bluff_v0.1.8_darwin_arm64.tar.gz"
      sha256 "54b97bc848a2a7d5037786ec08c7e271599517d07d8b6ac2a00fd39ed080d2cb"
    else
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.8/bluff_v0.1.8_darwin_amd64.tar.gz"
      sha256 "2fb18cf7bb4b31293e3877b246d38fc4ba019211e19a0f2ec9dafc443cc19988"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.8/bluff_v0.1.8_linux_arm64.tar.gz"
      sha256 "66f2d1c8a76908251ab1e9e9167f757528be4ec84301f52e579f62404a8fecb4"
    else
      url "https://github.com/thsnkhn/bluff/releases/download/v0.1.8/bluff_v0.1.8_linux_amd64.tar.gz"
      sha256 "dbeab4048c98e08ccdd1f6fbf8a4cc92fb529b6c3d9f6c91c6e4da67043a5242"
    end
  end

  def install
    bin.install "bluff"
  end

  test do
    assert_match "bluff v#{version}", shell_output("#{bin}/bluff --version")
  end
end
