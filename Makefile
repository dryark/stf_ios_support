MakefileERR := $(shell ./makefile_preflight.pl)
ifdef ERR
all: error
wdafree: error
error:
	$(error preflight errors)
else
all: wdafree wda bin/wda/web
wdafree: config.json\
 bin/coordinator\
 ios_video_stream\
 ios_video_pull\
 device_trigger\
 halias\
 wdaproxyalias\
 view_log\
 stf\
 ivf\
 devreset\
 runner\
 runnerdist\
 update_server\
 launchfolder\
 ve_alias \
 idalias
endif

.PHONY:\
 clone\
 stf\
 ios_video_stream\
 device_trigger\
 halias\
 hbin\
 wda\
 offline\
 coordinator\
 runnerdist\
 wdaproxyalias\
 wdaproxybin\
 devreset\
 ivf\
 launchfolder\
 updatedist\
 dist\
 pull\
 ou\
 ve_alias\
 wdafree\
 idalias\
 idbin

config.json:
	cp config.json.example config.json

# --- Special Commands ---

ou:
	$(eval CN=$(shell security find-identity -v -p codesigning | head -1 | cut -d\  -f 5-))
	@echo CN=$(CN)
	$(eval OU=$(shell security find-certificate -p -c $(CN) | openssl x509 -noout -subject | tr '/' '\n' | grep OU= | cut -d= -f 2 ))
	@echo OU=$(OU)
	@plutil -convert xml1 -o xcode.xml ~/Library/Preferences/com.apple.dt.Xcode.plist
	@cat xcode.xml | perl -0777 -ple 's/<data>(.+?)<\/data>/<string>$1<\/string>/gs;' > xcode_clean.xml
	@plutil -convert json -o xcode.json xcode_clean.xml
	$(eval XCODEUSER=$(shell jq '.DVTDeveloperAccountManagerAppleIDLists["IDE.Prod"][0].username' xcode.json -j))
	@echo Xcode user=$(XCODEUSER)
	$(eval XCODEOU=$(shell jq '.IDEProvisioningTeams["$(XCODEUSER)"][1].teamID' xcode.json -j))
	@echo OU of Xcode user=$(XCODEOU)

# --- LaunchAgents Folder ---

launchfolder: ~/Library/LaunchAgents

~/Library/LaunchAgents:
	@if [ ! -d ~/Library/LaunchAgents ]; then mkdir ~/Library/LaunchAgents; fi;

# --- VIDEO ENABLER ---

ve_alias: bin/video_enabler

bin/video_enabler: repos/ios_video_enabler/video_enabler repos/ios_video_enabler | repos/ios_video_enabler
	cp repos/ios_video_enabler/video_enabler bin/video_enabler

repos/ios_video_enabler/video_enabler: repos/ios_video_enabler repos/ios_video_enabler/main.m | repos/ios_video_enabler
	$(MAKE) -C repos/ios_video_enabler

repos/ios_video_enabler/main.m: | repos/ios_video_enabler

# --- DEVICE TRIGGER ---

device_trigger: bin/osx_ios_device_trigger

bin/osx_ios_device_trigger: repos/osx_ios_device_trigger repos/osx_ios_device_trigger/osx_ios_device_trigger/main.cpp
	$(MAKE) -C repos/osx_ios_device_trigger

repos/osx_ios_device_trigger/osx_ios_device_trigger/main.cpp: | repos/osx_ios_device_trigger

# --- VIEW LOG ---

view_log: view_log.go
	go install github.com/fsnotify/fsnotify@latest
	go install github.com/sirupsen/logrus@latest
	go build view_log.go

# --- H264_TO_JPEG ---

halias: bin/decode

bin/decode: repos/h264_to_jpeg/decode
	ln -f -s ../repos/h264_to_jpeg/ffmpeg bin/ffmpeg
	cp repos/h264_to_jpeg/decode bin/decode

hbin: repos/h264_to_jpeg/decode

repos/h264_to_jpeg/decode: repos/h264_to_jpeg repos/h264_to_jpeg/hw_decode.c repos/h264_to_jpeg/tracker.h | repos/h264_to_jpeg
	$(MAKE) -C repos/h264_to_jpeg

# --- IOS-DEPLOY ---

idalias: bin/ios-deploy

bin/ios-deploy: repos/ios-deploy/ios-deploy
	cp repos/ios-deploy/ios-deploy bin/ios-deploy

idbin: repos/ios-deploy/ios-deploy

repos/ios-deploy/ios-deploy: repos/ios-deploy repos/ios-deploy/ios-deploy.m | repos/ios-deploy
	$(MAKE) -C repos/ios-deploy

# --- IOS_AVF_PULL ---

ivf: bin/ivf_pull

bin/ivf_pull: repos/ios_avf_pull/ivf_pull
	cp repos/ios_avf_pull/ivf_pull bin/ivf_pull

repos/ios_avf_pull/ivf_pull: repos/ios_avf_pull repos/ios_avf_pull/ivf_pull.m repos/ios_avf_pull/uclop.h | repos/ios_avf_pull
	$(eval GIT_COMMIT=$(shell jq '.ios_avf_pull.commit' temp/current_versions.json -j))
	$(eval GIT_DATE=$(shell jq '.ios_avf_pull.date' temp/current_versions.json -j))
	$(eval GIT_REMOTE=$(shell jq '.ios_avf_pull.remote' temp/current_versions.json -j))
	$(eval EASY_VERSION=$(shell jq '.version' repos/ios_avf_pull/version.json -j))
	GIT_COMMIT="$(GIT_COMMIT)" GIT_DATE="$(GIT_DATE)" GIT_REMOTE="$(GIT_REMOTE)" EASY_VERSION="$(EASY_VERSION)" $(MAKE) -C repos/ios_avf_pull

# --- COORDINATOR ---

coordinator: bin/coordinator

coordinator_sources := $(wildcard coordinator/*.go)

bin/coordinator: $(coordinator_sources)
	$(eval GIT_COMMIT=$(shell jq '.ios_support.commit' temp/current_versions.json -j))
	$(eval GIT_DATE=$(shell jq '.ios_support.date' temp/current_versions.json -j))
	$(eval GIT_REMOTE=$(shell jq '.ios_support.remote' temp/current_versions.json -j))
	$(eval EASY_VERSION=$(shell jq '.version' version.json -j))
	GIT_COMMIT="$(GIT_COMMIT)" GIT_DATE="$(GIT_DATE)" GIT_REMOTE="$(GIT_REMOTE)" EASY_VERSION="$(EASY_VERSION)" $(MAKE) -C coordinator

# --- RUNNER ---

runner: runner/runner

runner_sources := $(wildcard runner/*.go)

runner/runner: $(runner_sources)
	$(eval GIT_COMMIT=$(shell jq '.ios_support.commit' temp/current_versions.json -j))
	$(eval GIT_DATE=$(shell jq '.ios_support.date' temp/current_versions.json -j))
	$(eval GIT_REMOTE=$(shell jq '.ios_support.remote' temp/current_versions.json -j))
	$(eval EASY_VERSION=$(shell jq '.version' version.json -j))
	GIT_COMMIT="$(GIT_COMMIT)" GIT_DATE="$(GIT_DATE)" GIT_REMOTE="$(GIT_REMOTE)" EASY_VERSION="$(EASY_VERSION)" $(MAKE) -C runner

# --- UPDATE SERVER ---

update_server: update_server/server

update_server/server: $(wildcard update_server/*.go)
	$(MAKE) -C update_server

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
#	cd repos/WebDriverAgent && ./Scripts/bootstrap.sh

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

# --- DevReset ---

devreset: bin/devreset

devreset_sources := $(wildcard repos/ios_video_pull/*.c)

repos/macos_usbdev_reset/devreset: repos/macos_usbdev_reset $(devreset_sources) | repos/macos_usbdev_reset
	$(MAKE) -C repos/macos_usbdev_reset

bin/devreset: repos/macos_usbdev_reset/devreset
	cp repos/macos_usbdev_reset/devreset bin/devreset

# --- REPO CLONES ---

pull:
	git -C repos/WebDriverAgent pull
	git -C repos/osx_ios_device_trigger pull
	git -C repos/stf-ios-provider pull
	git -C repos/ios_avf_pull pull
	git -C repo/ios_video_pull pull
	git -C repos/h264_to_jpeg pull
	git -C repos/wdaproxy pull
	git -C repos/libimobiledevice pull
	git -C repos/ios_video_enabler pull

clone: repos/WebDriverAgent repos/osx_ios_device_trigger repos/stf-ios-provider repos/ios_avf_pull repos/ios_video_pull repos/h264_to_jpeg repos/wdaproxy repos/libimobiledevice

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
	$(eval REPO=$(shell jq '.repo_wda // "https://github.com/appium/WebDriverAgent.git"' config.json -j))
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

repos/macos_usbdev_reset:
	git clone https://github.com/nanoscopic/macos_usbdev_reset.git repos/macos_usbdev_reset

repos/libimobiledevice:
	git clone https://github.com/libimobiledevice/libimobiledevice.git repos/libimobiledevice

repos/ios_avf_pull:
	git clone https://github.com/nanoscopic/ios_avf_pull.git repos/ios_avf_pull

repos/ios_video_enabler:
	git clone https://github.com/nanoscopic/ios_video_enabler.git repos/ios_video_enabler

repos/ios-deploy:
	git clone https://github.com/nanoscopic/ios-deploy.git repos/ios-deploy

# --- LibIMobileDevice ---

libimd: /usr/local/bin/ideviceinfo

/usr/local/bin/ideviceinfo: repos/libimobiledevice repos/libimobiledevice/tools/ideviceinfo | repos/libimobiledevice
	$(MAKE) -C repos/libimobiledevice install

repos/libimobiledevice/tools/ideviceinfo: repos/libimobiledevice repos/libimobiledevice/Makefile | repos/libimobiledevice
	$(MAKE) -C repos/libimobiledevice

repos/libimobiledevice/Makefile: | repos/libimobiledevice
	cd repos/libimobiledevice && NOCONFIGURE=1 ./autogen.sh
	cd repos/libimobiledevice && ./configure --disable-openssl

# --- STF ---

stf: repos/stf-ios-provider/package-lock.json

repos/stf-ios-provider/package-lock.json: repos/stf-ios-provider/package.json
	cd repos/stf-ios-provider && PATH="/usr/local/opt/node@19/bin:$(PATH)" npm install
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

# --- RUNNER DISTRIBUTION ---

runnerdist: runner.tgz

runnerfiles := \
	runner/gencert.pl \
	runner/runner \
	runner/runner.json

runner.tgz: runner/gencert.pl runner/runner runner/runner.json
	tar -h -czf runner.tgz $(runnerfiles)

# --- UPDATE SERVER DISTRIBUTION ---

updatedist: update_server.tgz

updatefiles := \
	update_server/server\
	update_server/updates/updates.json\
	update_server/updates/remote.pl\
	update_server/updates/index.html\
	update_server/updates/v1.json

update_server/updates/v1.tgz: dist.tgz
	cp dist.tgz update_server/updates/v1.tgz

update_server.tgz: $(updatefiles) update_server/updates/v1.tgz
	tar -h -czf update_server.tgz $(updatefiles) update_server/updates/v1.tgz

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
	logs/openvpn.log \
	build_info.json

offline/build_info.json: bin/coordinator bin/ios_video_stream bin/ivf_pull
	@./get-version-info.sh > offline/build_info.json

dist.tgz: offline/build_info.json ios_video_stream wda device_trigger halias bin/coordinator offline/repos/stf-ios-provider config.json view_log wdaproxyalias | offline/repos/stf-ios-provider
	@if [ ! -d offline/logs ]; then mkdir -p offline/logs; fi;
	@if [ ! -f offline/logs/openvpn.log ]; then touch offline/logs/openvpn.log; fi;
	tar -h -czf dist.tgz $(distfiles) -C offline $(offlinefiles)

clean: cleanstf cleanwda cleanlogs cleanivs cleanwdaproxy cleanrunner
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

cleanivf:
	$(MAKE) -C repos/ios_avf_pull clean
	$(RM) bin/ivf_pull

cleanwda:
	$(RM) -rf bin/wda
	cd repos/WebDriverAgent && xcodebuild -scheme WebDriverAgentRunner clean

cleanrunner:
	$(MAKE) -C runner clean

cleanlogs:
	$(RM) logs/*
	touch logs/.gitkeep
