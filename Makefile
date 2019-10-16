all: bin/coordinator video_enabler mirrorfeed device_trigger stf wda ffmpegalias wdaproxyalias

.PHONY: checkout stf video_enabler mirrorfeed device_trigger ffmpegalias ffmpegbin wda offline coordinator dist wdaproxyalias wdaproxybin
checkout: repos/stf repos/stf_ios_mirrorfeed repos/WebDriverAgent repos/osx_ios_device_trigger
stf: repos/stf/node_modules
video_enabler: bin/osx_ios_video_enabler
mirrorfeed: bin/stf_ios_mirrorfeed
device_trigger: bin/osx_ios_device_trigger
ffmpegalias: bin/ffmpeg
ffmpegbin: repos/ffmpeg/ffmpeg
wda: bin/wda/is_built
offline: offline/dist.tgz
dist: offline/dist.tgz
coordinator: bin/coordinator
wdaproxyalias: bin/wdaproxy
wdaproxybin: repos/wdaproxy/wdaproxy

bin/ffmpeg: | ffmpegbin
	@if [ -e bin/ffmpeg ]; then rm bin/ffmpeg; fi;
	cd bin &&	ln -s ../repos/ffmpeg/ffmpeg ffmpeg

bin/wdaproxy: | wdaproxybin
	@if [ -e bin/wdaproxy ]; then rm bin/wdaproxy; fi;
	cd bin &&	ln -s ../repos/wdaproxy/wdaproxy wdaproxy

repos/ffmpeg/ffmpeg: repos/ffmpeg
	cd repos/ffmpeg && ./configure  --prefix=/usr/local --enable-gpl --enable-nonfree --enable-libx264 --enable-libx265 --enable-libxvid
	$(MAKE) -C repos/ffmpeg

repos/wdaproxy/wdaproxy: repos/wdaproxy
	$(MAKE) -C repos/wdaproxy

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

repos/wdaproxy:
	git clone https://github.com/nanoscopic/wdaproxy.git repos/wdaproxy

offline/stf_ios_support.tgz:
	wget -O offline/stf_ios_support.tgz https://github.com/tmobile/stf_ios_support/archive/master.tar.gz

offline/repos/stf: repos/stf/node_modules
	mkdir -p offline/repos/stf
	rm offline/repos/stf/*
	ln -s ../../../repos/stf/node_modules      offline/repos/stf/node_modules
	ln -s ../../../repos/stf/package.json      offline/repos/stf/package.json
	ln -s ../../../repos/stf/package-lock.json offline/repos/stf/package-lock.json
	ln -s ../../../repos/stf/runmod.js         offline/repos/stf/runmod.js
	ln -s ../../../repos/stf/res               offline/repos/stf/res
	ln -s ../../../repos/stf/lib               offline/repos/stf/lib

config.json:
	cp config.json.example config.json

offline/dist.tgz: mirrorfeed wda device_trigger ffmpegalias bin/coordinator video_enabler offline/repos/stf config.json
	$(RM) bin/wda_is_built
	tar -h -czf offline/dist.tgz deps.rb init.sh bin/ config.json run -C offline repos/
	touch bin/wda_is_built

pipe:
	mkfifo pipe

clean:
	$(MAKE) -C coordinator clean
	$(MAKE) -C video_enabler clean
	$(RM) pipe

cleanstf:
	$(MAKE) -C repos/stf clean

bin/wda/is_built: repos/WebDriverAgent/WebDriverAgent.xcodeproj
	@if [ -e bin/wda ]; then rm -rf bin/wda; fi;
	mkdir -p bin/wda/Debug-iphoneos
	$(eval DEVID=$(shell jq .xcode_dev_team_id config.json -j))
	cd repos/WebDriverAgent && xcodebuild -scheme WebDriverAgentRunner -destination generic/platform=iOS CODE_SIGN_IDENTITY="iPhone Developer" DEVELOPMENT_TEAM="$(DEVID)" build-for-testing
	$(eval $(shell ./get-wda-build-path.sh)) # Spits out PROD_PATH
	cp -r $(PROD_PATH)/ bin/wda/
	touch bin/wda/is_built
