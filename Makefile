all: config.json bin/coordinator video_enabler mirrorfeed device_trigger stf wda ffmpegalias wdaproxyalias view_log app wda_wrapper

.PHONY:\
 checkout\
 stf\
 video_enabler\
 mirrorfeed\
 device_trigger\
 ffmpegalias\
 ffmpegbin\
 wda\
 offline\
 coordinator\
 dist\
 wdaproxyalias\
 wdaproxybin\
 app\
 icons\
 icns

config.json:
	cp config.json.example config.json

# --- DEVICE TRIGGER ---

device_trigger: bin/osx_ios_device_trigger

bin/osx_ios_device_trigger: repos/osx_ios_device_trigger repos/osx_ios_device_trigger/osx_ios_device_trigger/main.cpp
	$(MAKE) -C repos/osx_ios_device_trigger

# --- VIEW LOG ---

view_log: view_log.go
	go get github.com/fsnotify/fsnotify
	go get github.com/sirupsen/logrus
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

coordinator_sources := $(wildcard coordinator/*.go)

bin/coordinator: $(coordinator_sources)
	$(MAKE) -C coordinator

# --- WDAPROXY WRAPPER ---

wda_wrapper: bin/wda_wrapper

bin/wda_wrapper: wda_wrapper/wda_wrapper.go
	$(MAKE) -C wda_wrapper

# --- MIRROR FEED ---

mirrorfeed: bin/stf_ios_mirrorfeed

bin/stf_ios_mirrorfeed: repos/stf_ios_mirrorfeed repos/stf_ios_mirrorfeed/mirrorfeed/mirrorfeed.go
	$(MAKE) -C repos/stf_ios_mirrorfeed/mirrorfeed
	touch bin/stf_ios_mirrorfeed

# --- VIDEO ENABLER ---

video_enabler: bin/osx_ios_video_enabler

bin/osx_ios_video_enabler: video_enabler/Makefile
	$(MAKE) -C video_enabler

# --- WDA / WebDriverAgent ---

wdabootstrap: repos/WebDriverAgent/Carthage/Checkouts/RoutingHTTPServer

repos/WebDriverAgent/Carthage/Checkouts/RoutingHTTPServer: repos/WebDriverAgent
	cd repos/WebDriverAgent && ./Scripts/bootstrap.sh

wda: bin/wda/build_info.json

bin/wda/build_info.json: | wdabootstrap repos/WebDriverAgent repos/WebDriverAgent/WebDriverAgent.xcodeproj
	@if [ -e bin/wda ]; then rm -rf bin/wda; fi;
	@mkdir -p bin/wda/Debug-iphoneos
	ln -s ../../repos/wdaproxy/web bin/wda/web
	$(eval DEVID=$(shell jq .xcode_dev_team_id config.json -j))
	$(eval XCODEOPS=$(shell jq '.xcode_build_ops // ""' config.json -j))
	cd repos/WebDriverAgent && xcodebuild -scheme WebDriverAgentRunner -allowProvisioningUpdates -destination generic/platform=iOS $(XCODEOPS) CODE_SIGN_IDENTITY="iPhone Developer" DEVELOPMENT_TEAM="$(DEVID)" build-for-testing
	@# Spits out PROD_PATH
	$(eval $(shell ./get-wda-build-path.sh)) 
	@cp -r $(PROD_PATH)/ bin/wda/
	@./get-version-info.sh wda > bin/wda/build_info.json

# --- WDAProxy ---

wdaproxybin: repos/wdaproxy/wdaproxy

repos/wdaproxy/wdaproxy: repos/wdaproxy
	$(MAKE) -C repos/wdaproxy

wdaproxyalias: bin/wdaproxy

bin/wdaproxy: | wdaproxybin
	@if [ -e bin/wdaproxy ]; then rm bin/wdaproxy; fi;
	cd bin &&	ln -s ../repos/wdaproxy/wdaproxy wdaproxy

# --- REPO CLONES ---

checkout: repos/stf_ios_mirrorfeed repos/WebDriverAgent repos/osx_ios_device_trigger repos/stf-ios-provider

repos/stf-ios-provider:
	$(eval REPO=$(shell jq '.repo_stf // "https://github.com/nanoscopic/stf-ios-provider.git"' config.json -j))
	git clone $(REPO) repos/stf-ios-provider --branch master

repos/stf_ios_mirrorfeed:
	git clone https://github.com/tmobile/stf_ios_mirrorfeed.git repos/stf_ios_mirrorfeed

repos/WebDriverAgent:
	$(eval REPO=$(shell jq '.repo_wda // "https://github.com/appium/WebDriverAgent.git"' config.json -j))
	$(eval REPO_BR=$(shell jq '.repo_wda_branch // "master"' config.json -j))
	git clone $(REPO) repos/WebDriverAgent --branch $(REPO_BR)

repos/osx_ios_device_trigger:
	git clone https://github.com/tmobile/osx_ios_device_trigger.git repos/osx_ios_device_trigger

repos/ffmpeg:
	git clone https://github.com/nanoscopic/ffmpeg.git repos/ffmpeg

repos/wdaproxy:
	git clone https://github.com/nanoscopic/wdaproxy.git repos/wdaproxy

# --- OFFLINE STF ---

dist: offline/dist.tgz

offline/repos/stf: stf
	mkdir -p offline/repos/stf-ios-provider
	mkdir -p offline/logs
	rm offline/repos/stf/* & exit 0
	ln -s ../../../repos/stf-ios-provider/node_modules      offline/repos/stf-ios-provider/node_modules
	ln -s ../../../repos/stf-ios-provider/package.json      offline/repos/stf-ios-provider/package.json
	ln -s ../../../repos/stf-ios-provider/package-lock.json offline/repos/stf-ios-provider/package-lock.json
	ln -s ../../../repos/stf-ios-provider/runmod.js         offline/repos/stf-ios-provider/runmod.js
	ln -s ../../../repos/stf-ios-provider/lib               offline/repos/stf-ios-provider/lib
	ln -s ../../repos/wdaproxy/web             bin/wda/web & exit 0

# --- BINARY DISTRIBUTION ---

offline/dist.tgz: mirrorfeed wda device_trigger ffmpegalias bin/coordinator video_enabler repos/stf-ios-provider config.json view_log
	@./get-version-info.sh > offline/build_info.json
	tar -h -czf offline/dist.tgz video_pipes run stf_ios_support.rb *.sh view_log empty.tgz bin/ config.json -C offline repos/ build_info.json

pipe:
	mkfifo pipe

clean: cleanstf cleanwda cleanicon cleanlogs
	$(MAKE) -C coordinator clean
	$(MAKE) -C video_enabler clean
	$(RM) pipe build_info.json

cleanstf:
	$(MAKE) -C repos/stf clean

cleanwda:
	$(RM) -rf bin/wda

cleanapp:
	$(RM) -rf STF\ Coordinator.app

cleanicon:
	$(RM) -rf icon/stf.iconset icon/stf.iconset1 icon/stf.iconset2

cleanlogs:
	$(RM) logs/*
	touch logs/.gitkeep

# --- APP ---

app: STF\ Coordinator.app

STF\ Coordinator.app: | bin/coordinator icns
	mkdir -p STF\ Coordinator.app/Contents/MacOS
	mkdir -p STF\ Coordinator.app/Contents/Resources
	cp icon/stf.icns STF\ Coordinator.app/Contents/Resources/icon.icns
	cp bin/coordinator STF\ Coordinator.app/Contents/MacOS/
	cp config.json STF\ Coordinator.app/Contents/Resources/
	cp Info.plist STF\ Coordinator.app/Contents/
	./get-version-info.sh ios_support > STF\ Coordinator.app/Contents/Resources/build_info.json
	$(eval DEVID=$(shell jq .xcode_dev_team_id config.json -j))
	./util/signers.pl sign "$(DEVID)" "STF Coordinator.app"

icon/stf.iconset: | icon/stf.iconset1 icon/stf.iconset2 icons
	mkdir -p icon/stf.iconset
	cp icon/stf.iconset1/* icon/stf.iconset
	cp icon/stf.iconset2/* icon/stf.iconset

icon/stf.iconset1:
	mkdir icon/stf.iconset1

icon/stf.iconset2:
	mkdir icon/stf.iconset2

icon/stf.iconset: 

icons: \
 icon/stf.iconset1\
 icon/stf.iconset2\
 icon/stf.iconset1/icon_16x16.png\
 icon/stf.iconset2/icon_16x16@2x.png\
 icon/stf.iconset1/icon_32x32.png\
 icon/stf.iconset2/icon_32x32@2x.png\
 icon/stf.iconset1/icon_64x64.png\
 icon/stf.iconset2/icon_64x64@2x.png\
 icon/stf.iconset1/icon_128x128.png\
 icon/stf.iconset2/icon_128x128@2x.png\
 icon/stf.iconset1/icon_256x256.png\
 icon/stf.iconset2/icon_256x256@2x.png\
 icon/stf.iconset1/icon_512x512.png\
 icon/stf.iconset2/icon_512x512@2x.png\
 icon/stf.iconset1/icon_1024x1024.png

icon/stf.iconset1/icon_%.png: icon/stf_icon.png
	sips -z $(firstword $(subst x, ,$*)) $(firstword $(subst x, ,$*)) icon/stf_icon.png --out $@

icon/stf.iconset2/icon_%@2x.png: icon/stf_icon.png
	sips -z $$( echo $(firstword $(subst x, ,$*))*2 | bc) $$( echo $(firstword $(subst x, ,$*))*2 | bc) icon/stf_icon.png --out $@

icns: icon/stf.icns

icon/stf.icns: icon/stf.iconset
	iconutil -c icns -o icon/stf.icns icon/stf.iconset