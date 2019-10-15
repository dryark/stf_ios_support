all: wda

.PHONY: wda
wda: bin/wda_is_built

repos/WebDriverAgent/:
	mkdir repos/stf
	tar -C repos-stf --strip-components=1 -xf stf.tgz

bin/wda_is_built: repos/WebDriverAgent/WebDriverAgent.xcodeproj
	@if [ "${XCODE_DEVTEAM}" == "" ]; then echo "Must set env var XCODE_DEVTEAM"; exit 1; fi;
	cd repos/WebDriverAgent && carthage update --platform "iOS"
	cd repos/WebDriverAgent && xcodebuild -scheme WebDriverAgentRunner -destination generic/platform=iOS CODE_SIGN_IDENTITY="iPhone Developer" DEVELOPMENT_TEAM="$(XCODE_DEVTEAM)"
	touch bin/wda_is_built