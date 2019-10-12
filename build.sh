#!/bin/bash

function build_stf() {
  cd repos/stf
  make
  cd ../..
}

function build_mirrorfeed() {
  cd repos/stf_ios_mirrorfeed
  make
  cd ../..
}

function build_osx_ios_device_trigger() {
  cd repos/osx_ios_device_trigger
  make
  cd ../..
}

function build_osx_ios_video_enabler() {
  cd video_enabler
  make
  cd ..
}

function build_coordinator() {
  cd coordinator
  make
  cd ..
}

build_stf
build_mirrorfeed
build_osx_ios_device_trigger
build_osx_ios_video_enabler
build_coordinator