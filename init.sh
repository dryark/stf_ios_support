#!/bin/bash
mkdir -p repos

GR="\033[32m"
RED="\033[91m"
RST="\033[0m"

function assert_has_brew() {
  if ! command -v brew > /dev/null; then
    echo "Please make sure that you have homebrew installed (https://brew.sh/)"
    exit 1
  fi
}

function assert_has_node8() {
  NODE_MAJOR_VERSION="none"
  if command -v node > /dev/null; then
    NODE_VERSION=`node --version | tr -d "\n"`
    NODE_MAJOR_VERSION=`echo $NODE_VERSION | perl -pe 's/v([0-9]+).+/$1/'`
  fi
  
  if [ ! $NODE_MAJOR_VERSION == "8" ]; then
    echo -e "${RED}Node 8 not installed; installed node version: \"$NODE_MAJOR_VERSION\"$RST"
    exit 1
  else
    echo -e "${GR}Node 8 installed$RST"
  fi
}

function assert_has_xcodebuild() {
  XCODE_VERSION="none"
  XCODE_MAJOR_VERSION="0"
  XCODE_MINOR_VERSION="0"
  if command -v xcodebuild > /dev/null; then
    XCODE_VERSION=`xcodebuild -version | grep Xcode | tr -d "\n" | perl -pe 's/Xcode //'`
    XCODE_MAJOR_VERSION=`echo $XCODE_VERSION | perl -pe 's/([0-9]+)\.[0-9]+/$1/'`
    XCODE_MINOR_VERSION=`echo $XCODE_VERSION | perl -pe 's/[0-9]+\.([0-9]+)/$1/'`
  fi
  
  echo "XCODE Version: $XCODE_VERSION"
  echo "XCODE Version: Major = $XCODE_MAJOR_VERSION, Minor = $XCODE_MINOR_VERSION"
  
  if [ $XCODE_MAJOR_VERSION > 10 ]; then
    echo -e "${GR}Xcode $XCODE_VERSION installed$RST"
  elif [ "$XCODE_VERSION" == "10.3" ]; then
    echo -e "${GR}Xcode 10.3 installed$RST"
  else
    echo -e "${RED}Xcode 10.3+ not installed$RST"
    exit 1
  fi
}

function contained() {
  local e match="$1"
  shift
  for e; do [[ "$e" == "$match" ]] && return 0; done
  return 1
}

assert_has_brew
assert_has_node8
assert_has_xcodebuild

# STF dependencies
declare -a deps=("jq" "rethinkdb" "graphicsmagick" "zeromq" "protobuf" "yasm" "pkg-config" "carthage" "automake" "autoconf" "libtool" "wget" "libimobiledevice" "golang")

# Check and install dependencies
declare -a installed=( $(brew list) )
not_installed=()
i=0
for dependency in ${STF_DEPENDENCIES[@]}; do
  if ! contained ${dependency} "${installed[@]}"; then
    not_installed[$i]=${dependency}
    i=$(($i+1))
  fi
done
if [ ${#not_installed[@]} == 0 ]; then
  echo -e "${GR}All brew dependencies installed$RST"
else
  for lib in ${not_installed[@]}; do
    echo "Installing brew ${lib}"
    brew install ${lib}
  done
fi

