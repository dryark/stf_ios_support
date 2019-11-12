class StfIosSupport < Formula
  desc "OpenSTF IOS Device Provider"
  homepage ""
  url "https://github.com/nanoscopic/empty/archive/empty.tar.gz"
  version "1.0.0"
  sha256 "324c7d7662fd392fa2b7e0c9ce2bb9fd2ff677403c31311b13ac64bd1a15cbf7"

  def install
    system "touch #{prefix}/intentionally_empty_install"
  end

  # depends_on "cmake" => :build
  depends_on "jq"
  depends_on "rethinkdb"
  depends_on "graphicsmagick"
  depends_on "zeromq"
  depends_on "protobuf"
  depends_on "yasm"
  depends_on "pkg-config"
  depends_on "carthage"
  depends_on "automake"
  depends_on "autoconf"
  depends_on "libtool"
  depends_on "wget"
  # depends_on "libimobiledevice" # need to install with --HEAD
  depends_on "golang" => :build
  depends_on :xcode => "10.3"
  depends_on "node@8"
  depends_on "libsodium"
  depends_on "czmq"
  depends_on "sdl2"
  depends_on "x264"
  depends_on "x265"
end
