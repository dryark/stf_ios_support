all: config.json bin/coordinator video_enabler mirrorfeed device_trigger stf wda ffmpegalias wdaproxyalias view_log

.PHONY: checkout stf video_enabler mirrorfeed device_trigger ffmpegalias ffmpegbin wda offline coordinator dist wdaproxyalias wdaproxybin

config.json:
	cp config.json.example config.json

# --- DEVICE TRIGGER ---

device_trigger: bin/osx_ios_device_trigger

bin/osx_ios_device_trigger: | repos/osx_ios_device_trigger
	$(MAKE) -C repos/osx_ios_device_trigger

# --- VIEW LOG ---

view_log: view_log.go
	go build view_log.go

# --- FFMPEG ---

ffmpegalias: bin/ffmpeg

bin/ffmpeg: | ffmpegbin
	@if [ -e bin/ffmpeg ]; then rm bin/ffmpeg; fi;
	cd bin &&	ln -s ../repos/ffmpeg/ffmpeg ffmpeg

ffmpegbin: repos/ffmpeg/ffmpeg

repos/ffmpeg/ffmpeg: | repos/ffmpeg
	cd repos/ffmpeg && ./configure  --prefix=/usr/local --enable-gpl --enable-nonfree --enable-libx264 --enable-libx265 --enable-libxvid
	$(MAKE) -C repos/ffmpeg

# --- STF ---

stf: repos/stf/node_modules

repos/stf/node_modules: | repos/stf
	$(MAKE) -C repos/stf

# --- COORDINATOR ---

coordinator: bin/coordinator

bin/coordinator: coordinator/coordinator.go
	$(MAKE) -C coordinator

# --- MIRROR FEED ---

mirrorfeed: bin/stf_ios_mirrorfeed

bin/stf_ios_mirrorfeed: | repos/stf_ios_mirrorfeed repos/stf_ios_mirrorfeed/mirrorfeed/mirrorfeed.go
	$(MAKE) -C repos/stf_ios_mirrorfeed/mirrorfeed
	touch bin/stf_ios_mirrorfeed

# --- VIDEO ENABLER ---

video_enabler: bin/osx_ios_video_enabler

bin/osx_ios_video_enabler: video_enabler/Makefile
	$(MAKE) -C video_enabler

# --- WDA / WebDriverAgent ---

wdabootstrap: repos/WebDriverAgent/Carthage/Checkouts/CocoaAsyncSocket

repos/WebDriverAgent/Carthage/Checkouts/CocoaAsyncSocket: | repos/WebDriverAgent
	cd repos/WebDriverAgent && ./Scripts/bootstrap.sh

wda: bin/wda/is_built

bin/wda/is_built: | wdabootstrap repos/WebDriverAgent repos/WebDriverAgent/WebDriverAgent.xcodeproj
	@if [ -e bin/wda ]; then rm -rf bin/wda; fi;
	@mkdir -p bin/wda/Debug-iphoneos
	$(eval DEVID=$(shell jq .xcode_dev_team_id config.json -j))
	cd repos/WebDriverAgent && xcodebuild -scheme WebDriverAgentRunner -allowProvisioningUpdates -destination generic/platform=iOS CODE_SIGN_IDENTITY="iPhone Developer" DEVELOPMENT_TEAM="$(DEVID)" build-for-testing
	@# Spits out PROD_PATH
	$(eval $(shell ./get-wda-build-path.sh)) 
	@cp -r $(PROD_PATH)/ bin/wda/
	@touch bin/wda/is_built

# --- WDAProxy ---

wdaproxybin: repos/wdaproxy/wdaproxy

repos/wdaproxy/wdaproxy: repos/wdaproxy
	$(MAKE) -C repos/wdaproxy

wdaproxyalias: bin/wdaproxy

bin/wdaproxy: | wdaproxybin
	@if [ -e bin/wdaproxy ]; then rm bin/wdaproxy; fi;
	cd bin &&	ln -s ../repos/wdaproxy/wdaproxy wdaproxy

# --- REPO CLONES ---

checkout: repos/stf repos/stf_ios_mirrorfeed repos/WebDriverAgent repos/osx_ios_device_trigger

repos/stf:
	git clone https://github.com/tmobile/stf.git repos/stf --branch ios-support

repos/stf_ios_mirrorfeed:
	git clone https://github.com/tmobile/stf_ios_mirrorfeed.git repos/stf_ios_mirrorfeed

repos/WebDriverAgent:
	git clone https://github.com/petemyron/WebDriverAgent.git repos/WebDriverAgent --branch video-stream-control

repos/osx_ios_device_trigger:
	git clone https://github.com/tmobile/osx_ios_device_trigger.git repos/osx_ios_device_trigger

repos/ffmpeg:
	git clone https://github.com/nanoscopic/ffmpeg.git repos/ffmpeg

repos/wdaproxy:
	git clone https://github.com/nanoscopic/wdaproxy.git repos/wdaproxy

# --- OFFLINE STF ---

dist: offline/dist.tgz

offline/repos/stf: stf
	mkdir -p offline/repos/stf
	rm offline/repos/stf/* & exit 0
	ln -s ../../../repos/stf/node_modules      offline/repos/stf/node_modules
	ln -s ../../../repos/stf/package.json      offline/repos/stf/package.json
	ln -s ../../../repos/stf/package-lock.json offline/repos/stf/package-lock.json
	ln -s ../../../repos/stf/runmod.js         offline/repos/stf/runmod.js
	ln -s ../../../repos/stf/res               offline/repos/stf/res
	ln -s ../../../repos/stf/lib               offline/repos/stf/lib
	ln -s ../../repos/wdaproxy/web             bin/wda/web

# --- BINARY DISTRIBUTION ---

offline/dist.tgz: mirrorfeed wda device_trigger ffmpegalias bin/coordinator video_enabler offline/repos/stf config.json view_log
	$(RM) bin/wda_is_built
	tar -h -czf offline/dist.tgz run stf_ios_support.rb *.sh view_log empty.tgz bin/ config.json -C offline repos/
	touch bin/wda_is_built

pipe:
	mkfifo pipe

clean:
	$(MAKE) -C coordinator clean
	$(MAKE) -C video_enabler clean
	$(RM) pipe

cleanstf:
	$(MAKE) -C repos/stf clean

cleanwda:
	$(RM) bin/wda/is_built
