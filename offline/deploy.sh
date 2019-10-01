#!/bin/bash

INS=/Users/user/stf

mkdir -p $INS
cp *.tgz $INS
cd $INS
mkdir stf_ios_support
tar -C stf_ios_support --strip-components=1 -xf stf_ios_support.tgz
cd stf_ios_support
mkdir -p repos
cd repos

mkdir osx_ios_device_trigger
tar -C osx_ios_device_trigger --strip-components=1 -xf ../../osx_ios_device_trigger.tgz 

mkdir osx_ios_video_enabler
tar -C osx_ios_video_enabler --strip-components=1 -xf ../../osx_ios_video_enabler.tgz

mkdir stf
tar -C stf --strip-components=1 -xf ../../stf.tgz

mkdir stf_ios_mirrorfeed
tar -C stf_ios_mirrorfeed --strip-components=1 -xf ../../stf_ios_mirrorfeed.tgz

mkdir WebDriverAgent
tar -C WebDriverAgent --strip-components=1 -xf ../../wda.tgz

mkdir fake_libimobiledevice

# move back up to stf_ios_support dir
cd ..

# Install brew if it does not exist
if ! command -v brew > /dev/null; then
    /usr/bin/ruby -e "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/master/install)"
fi

brew install node@8
echo 'export PATH="/usr/local/opt/node@8/bin:$PATH"' >> ~/.bash_profile

brew install jq rethinkdb graphicsmagick zeromq protobuf yasm pkg-config carthage automake autoconf libtool wget libimobiledevice golang

# cxlab@t-mobile.com