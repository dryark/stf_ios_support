TARGET = ../bin/osx_ios_video_enabler

all: $(TARGET)

$(TARGET): osx_ios_video_enabler/main.m osx_ios_video_enabler.xcodeproj/project.pbxproj
	xcodebuild -scheme osx_ios_video_enabler TARGET_BUILD_DIR="../bin"

clean:
	$(RM) $(TARGET)