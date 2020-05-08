ERR := $(shell perl ./makefile_preflight.pl)
ifdef ERR
all: error
error:
	$(error preflight errors)
else
all: config.json bin/coordinator video_enabler ios_video_stream device_trigger wda halias wdaproxyalias view_log wda_wrapper stf bin/wda/web app
endif

.PHONY:\
 checkout\
 stf\
 video_enabler\
 ios_video_stream\
 device_trigger\
 halias\
 hbin\
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

# --- H264_TO_JPEG ---

halias: bin/decode

bin/decode: | hbin
	@if [ -e bin/deocde ]; then rm bin/decode; fi;
	ln -s ../repos/h264_to_jpeg/decode bin/decode
	ln -s ../repos/h264_to_jpeg/ffmpeg bin/ffmpeg

hbin: repos/h264_to_jpeg/decode

repos/h264_to_jpeg/decode: repos/h264_to_jpeg repos/h264_to_jpeg/hw_decode.c repos/h264_to_jpeg/tracker.h
	$(MAKE) -C repos/h264_to_jpeg

# --- COORDINATOR ---

coordinator: bin/coordinator

coordinator_sources := $(wildcard coordinator/*.go)

bin/coordinator: $(coordinator_sources)
	$(MAKE) -C coordinator

# --- WDAPROXY WRAPPER ---

wda_wrapper: bin/wda_wrapper

bin/wda_wrapper: wda_wrapper/wda_wrapper.go
	$(MAKE) -C wda_wrapper

# --- IOS VIDEO STREAM ---

ios_video_stream: bin/ios_video_stream

bin/ios_video_stream: repos/ios_video_stream
	$(MAKE) -C repos/ios_video_stream

# --- VIDEO ENABLER ---

video_enabler: bin/osx_ios_video_enabler

bin/osx_ios_video_enabler: video_enabler/Makefile
	$(MAKE) -C video_enabler

# --- WDA / WebDriverAgent ---

wdabootstrap: repos/WebDriverAgent/Carthage/Checkouts/RoutingHTTPServer

ifeq (,$(wildcard repos/WebDriverAgent/Carthage/Checkouts/RoutingHTTPServer))
repos/WebDriverAgent/Carthage/Checkouts/RoutingHTTPServer: repos/WebDriverAgent
	cd repos/WebDriverAgent && ./Scripts/bootstrap.sh
else
repos/WebDriverAgent/Carthage/Checkouts/RoutingHTTPServer: repos/WebDriverAgent
	@true
endif

wda: bin/wda/build_info.json

xcodebuildoptions1 := \
	-scheme WebDriverAgentRunner \
	-allowProvisioningUpdates \
	-destination generic/platform=iOS

DEVID := $(shell jq .xcode_dev_team_id config.json -j)

xcodebuildoptions2 := \
	CODE_SIGN_IDENTITY="iPhone Developer" \
	DEVELOPMENT_TEAM="$(DEVID)"

bin/wda/build_info.json: | wdabootstrap repos/WebDriverAgent repos/WebDriverAgent/WebDriverAgent.xcodeproj
	@if [ -e bin/wda ]; then rm -rf bin/wda; fi;
	@mkdir -p bin/wda/Debug-iphoneos
	ln -s ../../repos/wdaproxy/web bin/wda/web
	$(eval XCODEOPS=$(shell jq '.xcode_build_ops // ""' config.json -j))
	cd repos/WebDriverAgent && xcodebuild $(xcodebuildoptions1) $(XCODEOPS) $(xcodebuildoptions2) build-for-testing
	$(eval PROD_PATH=$(shell ./get-wda-build-path.sh))
	@if [ "$(PROD_PATH)" != "" ]; then cp -r $(PROD_PATH)/ bin/wda/; fi;
	@if [ "$(PROD_PATH)" != "" ]; then ./get-version-info.sh --repo wda > bin/wda/build_info.json; fi;
	@if [ "$(PROD_PATH)" == "" ]; then echo FAIL TO GET PRODUCTION PATH - you should rerun make; exit 1; fi;

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

repos/stf-ios-provider/package.json: repos/stf-ios-provider

repos/stf-ios-provider:
	$(eval REPO=$(shell jq '.repo_stf // "https://github.com/nanoscopic/stf-ios-provider.git"' config.json -j))
	git clone $(REPO) repos/stf-ios-provider --branch master

repos/ios_video_stream:
	git clone https://github.com/nanoscopic/ios_video_stream.git repos/ios_video_stream

repos/WebDriverAgent:
	$(eval REPO=$(shell jq '.repo_wda // "https://github.com/nanoscopic/WebDriverAgent.git"' config.json -j))
	$(eval REPO_BR=$(shell jq '.repo_wda_branch // "master"' config.json -j))
	git clone $(REPO) repos/WebDriverAgent --branch $(REPO_BR)

repos/osx_ios_device_trigger:
	git clone https://github.com/tmobile/osx_ios_device_trigger.git repos/osx_ios_device_trigger

repos/h264_to_jpeg:
	git clone https://github.com/nanoscopic/h264_to_jpeg.git repos/h264_to_jpeg

repos/wdaproxy:
	git clone https://github.com/nanoscopic/wdaproxy.git repos/wdaproxy

# --- STF ---

stf: repos/stf-ios-provider/package-lock.json

repos/stf-ios-provider/package-lock.json: repos/stf-ios-provider/package.json
	cd repos/stf-ios-provider && PATH=/usr/local/opt/node\@12/bin:$(PATH) npm install
	touch repos/stf-ios-provider/package-lock.json

# --- OFFLINE STF ---

dist: offline/dist.tgz

offline/repos/stf-ios-provider: repos/stf-ios-provider repos/stf-ios-provider/package-lock.json bin/wda/web
	mkdir -p offline/repos/stf-ios-provider
	rm -rf offline/repos/stf-ios-provider/*
	ln -s ../../../repos/stf-ios-provider/node_modules/     offline/repos/stf-ios-provider/node_modules
	ln -s ../../../repos/stf-ios-provider/package.json      offline/repos/stf-ios-provider/package.json
	ln -s ../../../repos/stf-ios-provider/package-lock.json offline/repos/stf-ios-provider/package-lock.json
	ln -s ../../../repos/stf-ios-provider/runmod.js         offline/repos/stf-ios-provider/runmod.js
	ln -s ../../../repos/stf-ios-provider/lib/              offline/repos/stf-ios-provider/lib

bin/wda/web:
	@if [ ! -L bin/wda/web ]; then ln -s ../../../repos/wdaproxy/web bin/wda/web; fi;

# --- BINARY DISTRIBUTION ---

distfiles := \
	run \
	stf_ios_support.rb \
	*.sh \
	view_log \
	bin/ \
	util/*.pl \
	config.json

offlinefiles := \
	repos/ \
	logs/ \
	build_info.json

offline/dist.tgz: ios_video_stream wda device_trigger halias bin/coordinator video_enabler offline/repos/stf-ios-provider config.json view_log
	@./get-version-info.sh > offline/build_info.json
	mkdir -p offline/logs
	touch offline/logs/openvpn.log
	tar -h -czf offline/dist.tgz $(distfiles) -C offline $(offlinefiles)

clean: cleanstf cleanwda cleanicon cleanlogs
	$(MAKE) -C coordinator clean
	$(MAKE) -C video_enabler clean
	$(RM) build_info.json

cleanstf:
	$(MAKE) -C repos/stf clean

cleanwda:
	$(RM) -rf bin/wda
	cd repos/WebDriverAgent && xcodebuild -scheme WebDriverAgentRunner clean

cleanapp:
	$(RM) -rf STF\ Coordinator.app

cleanicon:
	$(RM) -rf coordinator/icon/stf.iconset coordinator/icon/stf.iconset1 coordinator/icon/stf.iconset2

cleanlogs:
	$(RM) logs/*
	touch logs/.gitkeep

# --- APP ---

app: STF\ Coordinator.app

STF\ Coordinator.app: | bin/coordinator icns
	mkdir -p STF\ Coordinator.app/Contents/MacOS
	mkdir -p STF\ Coordinator.app/Contents/Resources
	cp coordinator/icon/stf.icns STF\ Coordinator.app/Contents/Resources/icon.icns
	cp bin/coordinator STF\ Coordinator.app/Contents/MacOS/
	$(eval CONFIGPATH=$(shell jq .install.config_path config.json -j))
	echo '{"config_path":"$(CONFIGPATH)"}' > STF\ Coordinator.app/Contents/Resources/config.json
	cp coordinator/app/Info.plist STF\ Coordinator.app/Contents/
	./get-version-info.sh --repo ios_support > STF\ Coordinator.app/Contents/Resources/build_info.json
	$(eval DEVID=$(shell jq .xcode_dev_team_id config.json -j))
	./util/signers.pl sign "$(DEVID)" "STF Coordinator.app"

coordinator/icon/stf.iconset: | coordinator/icon/stf.iconset1 coordinator/icon/stf.iconset2 icons
	mkdir -p coordinator/icon/stf.iconset
	cp coordinator/icon/stf.iconset1/* coordinator/icon/stf.iconset
	cp coordinator/icon/stf.iconset2/* coordinator/icon/stf.iconset

coordinator/icon/stf.iconset1:
	mkdir coordinator/icon/stf.iconset1

coordinator/icon/stf.iconset2:
	mkdir coordinator/icon/stf.iconset2

coordinator/icon/stf.iconset: 

icons: \
 coordinator/icon/stf.iconset1\
 coordinator/icon/stf.iconset2\
 coordinator/icon/stf.iconset1/icon_16x16.png\
 coordinator/icon/stf.iconset2/icon_16x16@2x.png\
 coordinator/icon/stf.iconset1/icon_32x32.png\
 coordinator/icon/stf.iconset2/icon_32x32@2x.png\
 coordinator/icon/stf.iconset1/icon_64x64.png\
 coordinator/icon/stf.iconset2/icon_64x64@2x.png\
 coordinator/icon/stf.iconset1/icon_128x128.png\
 coordinator/icon/stf.iconset2/icon_128x128@2x.png\
 coordinator/icon/stf.iconset1/icon_256x256.png\
 coordinator/icon/stf.iconset2/icon_256x256@2x.png\
 coordinator/icon/stf.iconset1/icon_512x512.png\
 coordinator/icon/stf.iconset2/icon_512x512@2x.png\
 coordinator/icon/stf.iconset1/icon_1024x1024.png

coordinator/icon/stf.iconset1/icon_%.png: coordinator/icon/stf_icon.png
	sips -z $(firstword $(subst x, ,$*)) $(firstword $(subst x, ,$*)) coordinator/icon/stf_icon.png --out $@

coordinator/icon/stf.iconset2/icon_%@2x.png: coordinator/icon/stf_icon.png
	sips -z $$( echo $(firstword $(subst x, ,$*))*2 | bc) $$( echo $(firstword $(subst x, ,$*))*2 | bc) coordinator/icon/stf_icon.png --out $@

icns: coordinator/icon/stf.icns

coordinator/icon/stf.icns: coordinator/icon/stf.iconset
	iconutil -c icns -o coordinator/icon/stf.icns coordinator/icon/stf.iconset
