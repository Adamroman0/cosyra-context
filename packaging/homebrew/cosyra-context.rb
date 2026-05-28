class CosyraContext < Formula
  desc "Minimal project-local context bridge for AI coding tools"
  homepage "https://github.com/Adamroman0/cosyra-context"
  version "0.1.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://github.com/Adamroman0/cosyra-context/releases/download/v0.1.0/cosyra-darwin-arm64.tar.gz"
      sha256 "REPLACE_WITH_SHA256"
    else
      url "https://github.com/Adamroman0/cosyra-context/releases/download/v0.1.0/cosyra-darwin-amd64.tar.gz"
      sha256 "REPLACE_WITH_SHA256"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://github.com/Adamroman0/cosyra-context/releases/download/v0.1.0/cosyra-linux-arm64.tar.gz"
      sha256 "REPLACE_WITH_SHA256"
    else
      url "https://github.com/Adamroman0/cosyra-context/releases/download/v0.1.0/cosyra-linux-amd64.tar.gz"
      sha256 "REPLACE_WITH_SHA256"
    end
  end

  def install
    bin.install "cosyra"
  end

  test do
    system "#{bin}/cosyra", "version"
  end
end

