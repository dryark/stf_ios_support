#!/bin/bash

mkdir offline
cd offline
if [ ! -f stf_ios_support.tgz ]; then
    wget -O stf_ios_support.tgz https://github.com/nanoscopic/stf_ios_support/archive/master.tar.gz
fi

#git clone https://github.com/nanoscopic/fake_libimobiledevice.git repos/fake_libimobiledevice

if [ ! -f stf.tgz ]; then
    wget -O stf.tgz https://github.com/nanoscopic/stf/archive/ios-support.tar.gz
fi

if [ ! -f stf_ios_mirrorfeed.tgz ]; then
    wget -O stf_ios_mirrorfeed.tgz https://github.com/nanoscopic/stf_ios_mirrorfeed/archive/master.tar.gz
fi

if [ ! -f wda.tgz ]; then
    wget -O wda.tgz https://github.com/nanoscopic/WebDriverAgent/archive/video-stream-control.tar.gz
fi

if [ ! -f osx_ios_device_trigger.tgz ]; then
    wget -O osx_ios_device_trigger.tgz https://github.com/nanoscopic/osx_ios_device_trigger/archive/master.tar.gz
fi

if [ ! -f osx_ios_video_enabler.tgz ]; then
    wget -O osx_ios_video_enabler.tgz https://github.com/nanoscopic/osx_ios_video_enabler/archive/master.tar.gz
fi