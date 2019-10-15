all: bin/coordinator video_enabler mirrorfeed device_trigger stf wda ffmpegalias

.PHONY: checkout stf video_enabler mirrorfeed device_trigger ffmpegalias ffmpegbin wda offline coordinator
checkout: repos/stf repos/stf_ios_mirrorfeed repos/WebDriverAgent repos/osx_ios_device_trigger
stf: repos/stf/node_modules
video_enabler: bin/osx_ios_video_enabler
mirrorfeed: bin/stf_ios_mirrorfeed
device_trigger: bin/osx_ios_device_trigger
ffmpegalias: bin/ffmpeg
ffmpegbin: repos/ffmpeg/ffmpeg
wda: bin/wda_is_built
offline: offline/dist.tgz
coordinator: bin/coordinator

bin/ffmpeg: | ffmpegbin
	@if [ -e bin/ffmpeg ]; then rm bin/ffmpeg; fi;
	cd bin &&	ln -s ../repos/ffmpeg/ffmpeg ffmpeg

repos/ffmpeg/ffmpeg: repos/ffmpeg
	cd repos/ffmpeg && ./configure  --prefix=/usr/local --enable-gpl --enable-nonfree --enable-libx264 --enable-libx265 --enable-libxvid
	$(MAKE) -C repos/ffmpeg

repos/stf/node_modules: | repos/stf
	$(MAKE) -C repos/stf

bin/coordinator: coordinator/coordinator.go
	$(MAKE) -C coordinator

bin/stf_ios_mirrorfeed: repos/stf_ios_mirrorfeed/mirrorfeed/mirrorfeed.go | repos/stf_ios_mirrorfeed
	$(MAKE) -C repos/stf_ios_mirrorfeed/mirrorfeed
	touch bin/stf_ios_mirrorfeed

bin/osx_ios_device_trigger:
	$(MAKE) -C repos/osx_ios_device_trigger

bin/osx_ios_video_enabler:
	$(MAKE) -C video_enabler

repos/stf:
	git clone https://github.com/tmobile/stf.git repos/stf --branch ios-support

repos/stf_ios_mirrorfeed:
	git clone https://github.com/tmobile/stf_ios_mirrorfeed.git repos/stf_ios_mirrorfeed

offline/stf_ios_mirrorfeed.tgz:
	wget -O offline/stf_ios_mirrorfeed.tgz https://github.com/tmobile/stf_ios_mirrorfeed/archive/master.tar.gz

repos/WebDriverAgent:
	git clone https://github.com/tmobile/WebDriverAgent.git repos/WebDriverAgent --branch video-stream-control

offline/wda.tgz:
	wget -O wda.tgz https://github.com/tmobile/WebDriverAgent/archive/video-stream-control.tar.gz

repos/osx_ios_device_trigger:
	git clone https://github.com/tmobile/osx_ios_device_trigger.git repos/osx_ios_device_trigger

offline/device_trigger.tgz:
	wget -O device_trigger.tgz https://github.com/tmobile/osx_ios_device_trigger/archive/master.tar.gz

repos/ffmpeg:
	git clone https://github.com/nanoscopic/ffmpeg.git repos/ffmpeg

offline/stf_ios_support.tgz:
	wget -O offline/stf_ios_support.tgz https://github.com/tmobile/stf_ios_support/archive/master.tar.gz

offline/repos/stf: repos/stf/node_modules
	mkdir -p offline/repos/stf
	ln -s ../../../repos/stf/node_modules      offline/repos/stf/node_modules
	ln -s ../../../repos/stf/package.json      offline/repos/stf/package.json
	ln -s ../../../repos/stf/package-lock.json offline/repos/stf/package-lock.json
	ln -s ../../../repos/stf/runmod.js         offline/repos/stf/runmod.js
	ln -s ../../../repos/stf/res               offline/repos/stf/res
	ln -s ../../../repos/stf/lib               offline/repos/stf/lib

offline/dist.tgz: mirrorfeed offline/wda.tgz device_trigger ffmpegalias bin/coordinator video_enabler offline/repos/stf
	$(RM) bin/wda_is_built
	tar -h -czf offline/dist.tgz deps.rb init.sh bin/ -C offline repos/ wda.tgz Makefile
	touch bin/wda_is_built

pipe:
	mkfifo pipe

clean:
	$(MAKE) -C coordinator clean
	$(MAKE) -C video_enabler clean
	$(RM) pipe

cleanstf:
	$(MAKE) -C repos/stf clean

bin/wda_is_built: repos/WebDriverAgent/WebDriverAgent.xcodeproj
	@if [ "${XCODE_DEVTEAM}" == "" ]; then echo "Must set env var XCODE_DEVTEAM"; exit 1; fi;
	cd repos/WebDriverAgent && carthage update --platform "iOS"
	cd repos/WebDriverAgent && xcodebuild -scheme WebDriverAgentRunner -destination generic/platform=iOS CODE_SIGN_IDENTITY="iPhone Developer" DEVELOPMENT_TEAM="${XCODE_DEVTEAM}"
	touch bin/wda_is_built