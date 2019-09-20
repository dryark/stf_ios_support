#!/bin/bash
mkdir -p repos

GR="\033[32m"
RED="\033[91m"
RST="\033[0m"

if [ ! -d "repos/fake_libimobiledevice" ]; then
	git clone https://github.com/nanoscopic/fake_libimobiledevice.git repos/fake_libimobiledevice
else
    echo -e "${GR}repos/fake_libimobiledevice exists$RST"
fi

if [ ! -d "repos/stf" ]; then
	git clone https://github.com/nanoscopic/stf.git repos/stf --branch ios-support
else
    echo -e "${GR}repos/stf exists$RST"
fi

if [ ! -d "repos/stf_ios_mirrorfeed" ]; then
	git clone https://github.com/nanoscopic/stf_ios_mirrorfeed.git repos/stf_ios_mirrorfeed
else
    echo -e "${GR}repos/stf_ios_mirrorfeed exists$RST"
fi

if [ ! -d "repos/WebDriverAgent" ]; then
	git clone https://github.com/nanoscopic/WebDriverAgent.git repos/WebDriverAgent --branch video-stream-control
else
    echo -e "${GR}repos/WebDriverAgent exists$RST"
fi

if [ ! -d "repos/osx_ios_device_trigger" ]; then
	git clone https://github.com/nanoscopic/osx_ios_device_trigger.git repos/osx_ios_device_trigger
else
    echo -e "${GR}repos/osx_ios_device_trigger exists$RST"
fi

if [ ! -d "repos/osx_ios_video_enabler" ]; then
	git clone https://github.com/nanoscopic/osx_ios_video_enabler.git repos/osx_ios_video_enabler
else
    echo -e "${GR}repos/osx_ios_video_enabler exists$RST"
fi

if [ ! -p "pipe" ]; then
    echo "Pipe does not exist; creating"
    mkfifo pipe
else
    echo -e "${GR}Pipe exists$RST"
fi

# STF dependencies
STF_DEPENDENCIES=("jq" "rethinkdb" "graphicsmagick" "zeromq" "protobuf" "yasm" "pkg-config" "carthage" "automake" "autoconf" "libtool" "wget" "libimobiledevice")

# --- FUNCTIONS ---

function in_array() {
  local array=${1}[@]
  local needle=${2}
  for i in ${!array}; do
    if [[ ${i} == ${needle} ]]; then
        return 0
    fi
  done
  return 1
}

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
  if command -v xcodebuild > /dev/null; then
    XCODE_VERSION=`xcodebuild -version | grep Xcode | tr -d "\n"`
  fi
  
  if [ ! "$XCODE_VERSION" == "Xcode 10.3" ]; then
    echo -e "${RED}Xcode 10.3 not installed$RST"
    exit 1
  else
    echo -e "${GR}Xcode 10.3 installed$RST"
  fi
}

assert_has_brew
assert_has_node8
assert_has_xcodebuild

# Check and install dependencies
installed=( $(brew list) )
not_installed=()
i=0
for dependency in ${STF_DEPENDENCIES[@]}; do
  if ! in_array installed ${dependency}; then
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
