#!/bin/bash
mkdir -p repos
git clone https://github.com/nanoscopic/fake_libimobiledevice.git repos/fake_libimobiledevice
git clone https://github.com/nanoscopic/stf.git repos/stf --branch ios-support
git clone https://github.com/nanoscopic/stf_ios_mirrorfeed.git repos/stf_ios_mirrorfeed
git clone https://github.com/nanoscopic/WebDriverAgent.git repos/WebDriverAgent --branch video-stream-control
git clone https://github.com/nanoscopic/osx_ios_device_trigger.git repos/osx_ios_device_trigger
git clone https://github.com/nanoscopic/osx_ios_video_enabler.git repos/osx_ios_video_enabler
mkfifo pipe

# STF dependencies
STF_DEPENDENCIES=("jq" "rethinkdb" "graphicsmagick" "zeromq" "protobuf" "yasm" "pkg-config" "carthage" "automake" "autoconf" "libtool" "wget" "libimobiledevice")

assert_has_brew

# function, check element in array
in_array() {
  local array=${1}[@]
  local needle=${2}
  for i in ${!array}; do
    if [[ ${i} == ${needle} ]]; then
        return 0
    fi
  done
  return 1
}


# Check and install dependencies
echo "Check dependencies:"
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
  echo "All dependencies installed"
else
  for lib in ${not_installed[@]}; do
    echo "Install ${lib}"
    brew install ${lib}
  done
fi

function assert_has_brew() {
  if ! command -v node > /dev/null; then
    echo "Please make sure that you have homebrew installed (https://brew.sh/)"
    exit 1
  fi
}
