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
  # depends_on "rethinkdb"
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
  depends_on "go" => :build
  depends_on "ossp-uuid"
  depends_on "socat"
  depends_on :xcode => "10.3"
  depends_on "node@14"
  depends_on "libsodium"
  depends_on "czmq"
  depends_on "jpeg-turbo"
  depends_on "nanomsg"
  depends_on "libgcrypt"
  depends_on "gnutls"
  depends_on "mobiledevice"
  # depends_on "libplist" # need to install with --HEAD
  # depends_on "libusbmuxd" # need to install with --HEAD
end
