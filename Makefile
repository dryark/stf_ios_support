MakefileERR := $(shell ./makefile_preflight.pl)
ifdef ERR
all: error
error:
	$(error preflight errors)
else
all: config.json bin/coordinator ios_video_stream ios_video_pull device_trigger wda halias wdaproxyalias view_log wda_wrapper stf bin/wda/web
endif

.PHONY:\
 checkout\
 stf\
 ios_video_stream\
 device_trigger\
 halias\
 hbin\
 wda\
 offline\
 coordinator\
 dist\
 wdaproxyalias\
 wdaproxybin

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

bin/decode: repos/h264_to_jpeg/decode
	ln -f -s ../repos/h264_to_jpeg/ffmpeg bin/ffmpeg
	cp repos/h264_to_jpeg/decode bin/decode

hbin: repos/h264_to_jpeg/decode

repos/h264_to_jpeg/decode: repos/h264_to_jpeg/hw_decode.c repos/h264_to_jpeg/tracker.h | repos/h264_to_jpeg
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

ivs_sources := $(wildcard repos/ios_video_stream/*.go)

repos/ios_video_stream/ios_video_stream: repos/ios_video_stream $(ivs_sources) | repos/ios_video_stream
	$(MAKE) -C repos/ios_video_stream

bin/ios_video_stream: repos/ios_video_stream/ios_video_stream
	cp repos/ios_video_stream/ios_video_stream bin/ios_video_stream

# --- IOS VIDEO PULL ---

ios_video_pull: bin/ios_video_pull

ivp_sources := $(wildcard repos/ios_video_pull/*.go)

repos/ios_video_pull/ios_video_pull: repos/ios_video_pull $(ivp_sources) | repos/ios_video_pull
	$(MAKE) -C repos/ios_video_pull

bin/ios_video_pull: repos/ios_video_pull/ios_video_pull
	cp repos/ios_video_pull/ios_video_pull bin/ios_video_pull

# --- WDA / WebDriverAgent ---

repos/WebDriverAgent/Carthage/Checkouts/RoutingHTTPServer/Info.plist: | repos/WebDriverAgent
	cd repos/WebDriverAgent && ./Scripts/bootstrap.sh

wda: bin/wda/build_info.json

xcodebuildoptions1 := \
	-scheme WebDriverAgentRunner \
	-allowProvisioningUpdates \
	-destination generic/platform=iOS

DEVID := $(shell jq .xcode_dev_team_id config.json -j)

xcodebuildoptions2 := \
	CODE_SIGN_IDENTITY="iPhone Developer" \
	DEVELOPMENT_TEAM="$(DEVID)"

bin/wda/build_info.json: repos/WebDriverAgent repos/WebDriverAgent/WebDriverAgent.xcodeproj | repos/WebDriverAgent repos/WebDriverAgent/Carthage/Checkouts/RoutingHTTPServer/Info.plist
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

repos/wdaproxy/wdaproxy: repos/wdaproxy repos/wdaproxy/main.go | repos/wdaproxy
	$(MAKE) -C repos/wdaproxy

wdaproxyalias: bin/wdaproxy

bin/wdaproxy: repos/wdaproxy/wdaproxy
	cp repos/wdaproxy/wdaproxy bin/wdaproxy

# --- REPO CLONES ---

checkout: repos/stf_ios_mirrorfeed repos/WebDriverAgent repos/osx_ios_device_trigger repos/stf-ios-provider

repos/stf-ios-provider/package.json: repos/stf-ios-provider

repos/stf-ios-provider:
	$(eval REPO=$(shell jq '.repo_stf // "https://github.com/nanoscopic/stf-ios-provider.git"' config.json -j))
	git clone $(REPO) repos/stf-ios-provider --branch master

repos/ios_video_stream:
	git clone https://github.com/nanoscopic/ios_video_stream.git repos/ios_video_stream

repos/ios_video_pull:
	git clone https://github.com/nanoscopic/ios_video_pull.git repos/ios_video_pull

repos/WebDriverAgent/WebDriverAgent.xcodeproj: repos/WebDriverAgent

repos/WebDriverAgent:
	$(eval REPO=$(shell jq '.repo_wda // "https://github.com/nanoscopic/WebDriverAgent.git"' config.json -j))
	$(eval REPO_BR=$(shell jq '.repo_wda_branch // "master"' config.json -j))
	git clone $(REPO) repos/WebDriverAgent --branch $(REPO_BR)

repos/osx_ios_device_trigger:
	git clone https://github.com/tmobile/osx_ios_device_trigger.git repos/osx_ios_device_trigger

repos/h264_to_jpeg/hw_decode.c: repos/h264_to_jpeg
repos/h264_to_jpeg/tracker.h: repos/h264_to_jpeg

repos/h264_to_jpeg:
	git clone https://github.com/nanoscopic/h264_to_jpeg.git repos/h264_to_jpeg

repos/wdaproxy/main.go: repos/wdaproxy	

repos/wdaproxy:
	git clone https://github.com/nanoscopic/wdaproxy.git repos/wdaproxy

# --- STF ---

stf: repos/stf-ios-provider/package-lock.json

repos/stf-ios-provider/package-lock.json: repos/stf-ios-provider/package.json
	cd repos/stf-ios-provider && PATH=/usr/local/opt/node\@12/bin:$(PATH) npm install
	touch repos/stf-ios-provider/package-lock.json

# --- OFFLINE STF ---

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

dist: dist.tgz

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

dist.tgz: ios_video_stream wda device_trigger halias bin/coordinator offline/repos/stf-ios-provider config.json view_log wdaproxyalias
	@./get-version-info.sh > offline/build_info.json
	mkdir -p offline/logs
	touch offline/logs/openvpn.log
	tar -h -czf dist.tgz $(distfiles) -C offline $(offlinefiles)

clean: cleanstf cleanwda cleanlogs cleanivs cleanwdaproxy
	$(MAKE) -C coordinator clean
	$(RM) build_info.json

cleanwdaproxy:
	$(MAKE) -C repos/wdaproxy clean
	$(RM) bin/wdaproxy

cleanstf:
	$(MAKE) -C repos/stf clean

cleanivs:
	$(MAKE) -C repos/ios_video_stream clean
	$(RM) bin/ios_video_stream

cleanwda:
	$(RM) -rf bin/wda
	cd repos/WebDriverAgent && xcodebuild -scheme WebDriverAgentRunner clean

cleanlogs:
	$(RM) logs/*
	touch logs/.gitkeep
