#!/bin/sh
# This script config params, check dependency command tools installed, and check all project exist
# Please config DEVELOPMENT_TEAM WDA_BUNDLE_ID according to your xcode setting


# your development team id of xcode
DEVELOPMENT_TEAM="[your devteam id]"
# unique bundle ID of WebDriverAgent
WDA_BUNDLE_ID="com.facebook.WebDriverAgentLib"

# directory of OPENSTF project
#OPENSTF_HOME="$(realpath $(dirname $0)/../)"
OPENSTF_HOME="/Users/davidh/git/openstf-ios-extended/"
# STF dependencies
STF_DEPENDENCIES=("rethinkdb" "graphicsmagick" "zeromq" "protobuf" "yasm" "pkg-config" "carthage" "automake" "autoconf" "libtool" "wget" "libimobiledevice")

function assert_has_brew() {
  if ! command -v node > /dev/null; then
    echo "Please make sure that you have homebrew installed (https://brew.sh/)"
    exit 1
  fi
}

function assert_has_node() {
  if ! command -v node > /dev/null; then
    echo "Please make sure that you have [node.js 8+] installed (https://nodejs.org/)"
    exit 1
  fi
}

function assert_has_xcodebuild() {
  if ! command -v xcodebuild > /dev/null; then
    echo "Please make sure that you have [Xcode 10.0+] and Xcode command line tools installed (https://developer.apple.com/xcode/)"
    exit 1
  fi
}


assert_has_brew
assert_has_node
assert_has_xcodebuild

if [ ! -d "${OPENSTF_HOME}" ]; then
  echo "Directory ${OPENSTF_HOME} not exist! Please set OPENSTF_HOME to the directory of project openstf in config.sh"
  exit 1
fi
if [ ! -f "${OPENSTF_HOME}/stf/package.json" ]; then
  echo "Project stf in directory ${OPENSTF_HOME} not exist!"
  exit 1
fi
if [ ! -d "${OPENSTF_HOME}/WebDriverAgent/WebDriverAgent.xcodeproj" ]; then
  echo "Project WebDriverAgent in directory ${OPENSTF_HOME} not exist!"
  exit 1
fi
