all: bin/coordinator video_enabler mirrorfeed device_trigger stf wda ffmpegalias

bin/coordinator:
	$(MAKE) -C coordinator

bin/osx_ios_video_enabler:
	$(MAKE) -C video_enabler

.PHONY: checkout stf video_enabler mirrorfeed device_trigger ffmpegalias ffmpegbin wda
checkout: repos/stf repos/stf_ios_mirrorfeed repos/WebDriverAgent repos/osx_ios_device_trigger
stf: repos/stf/node_modules
video_enabler: bin/osx_ios_video_enabler
mirrorfeed: bin/stf_ios_mirrorfeed
device_trigger: bin/osx_ios_device_trigger
ffmpegalias: bin/ffmpeg
ffmpegbin: repos/ffmpeg/bin/ffmpeg
wda: bin/wda_is_built

bin/ffmpeg: ffmpegbin
	cd bin &&	ln -s ../repos/ffmpeg/bin/ffmpeg ffmpeg

repos/ffmpeg/bin/ffmpeg:
	cd repos/ffmpeg && ./configure
	$(MAKE) -C repos/ffmpeg

bin/stf/node_modules:
	$(MAKE) -C repos/stf

bin/stf_ios_mirrorfeed:
	$(MAKE) -C repos/stf_ios_mirrorfeed/mirrorfeed

bin/osx_ios_device_trigger:
	$(MAKE) -C repos/osx_ios_device_trigger

repos/stf:
	git clone https://github.com/nanoscopic/stf.git repos/stf --branch ios-support

repos/stf_ios_mirrorfeed:
	git clone https://github.com/nanoscopic/stf_ios_mirrorfeed.git repos/stf_ios_mirrorfeed

repos/WebDriverAgent:
	git clone https://github.com/nanoscopic/WebDriverAgent.git repos/WebDriverAgent --branch video-stream-control

repos/osx_ios_device_trigger:
	git clone https://github.com/nanoscopic/osx_ios_device_trigger.git repos/osx_ios_device_trigger

repos/ffmpeg:
	git clone https://github.com/nanoscopic/ffmpeg.git repos/ffmpeg

pipe:
	mkfifo pipe

clean:
	$(MAKE) -C coordinator clean
	$(MAKE) -C video_enabler clean
	$(RM) pipe

cleanstf:
	$(MAKE) -C repos/stf clean

bin/wda_is_built: repos/WebDriverAgent/WebDriverAgent.xcodeproj
	cd repos/WebDriverAgent && carthage update --platform "iOS"
	cd repos/WebDriverAgent && xcodebuild -scheme WebDriverAgentRunner -destination generic/platform=iOS CODE_SIGN_IDENTITY="iPhone Developer" DEVELOPMENT_TEAM="$(XCODE_DEVTEAM)"
	touch bin/wda_is_built

clean:
	$(RM) $(TARGET)