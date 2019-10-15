class Deps < Formula
  desc "OpenSTF IOS Device Provider"
  homepage ""
  url "https://github.com/nanoscopic/empty/archive/empty.tar.gz"
  version "1.0.0"
  sha256 "548a04aaa960c0c9e9067e71a5b480d12f58325a34bf3c7774fda6682247da9a"

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
  depends_on "libimobiledevice"
  depends_on "golang" => :build
  depends_on :xcode => "10.3"
  depends_on "node" => "8"
end
