#!/bin/bash
mkdir -p repos
git clone https://github.com/nanoscopic/fake_libimobiledevice.git repos/fake_libimobiledevice
git clone https://github.com/nanoscopic/stf.git repos/stf --branch ios-support
git clone https://github.com/nanoscopic/stf_ios_mirrorfeed.git repos/stf_ios_mirrorfeed
git clone https://github.com/nanoscopic/WebDriverAgent.git repos/WebDriverAgent --branch video-stream-control
git clone https://github.com/nanoscopic/osx_ios_device_trigger.git repos/osx_ios_device_trigger
mkfifo pipe
