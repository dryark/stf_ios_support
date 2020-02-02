#!/bin/bash
# From https://stackoverflow.com/questions/3915040/bash-fish-command-to-print-absolute-path-to-a-file
abspath() {
    if [ -d "$1" ]; then (cd "$1"; pwd)
    elif [ -f "$1" ]; then
        if [[ $1 = /* ]];     then echo "$1"
        elif [[ $1 == */* ]]; then echo "$(cd "${1%/*}"; pwd)/${1##*/}"
        else                       echo "$(pwd)/$1"; fi
    fi
}

BPATH=$(xcodebuild -project repos/WebDriverAgent/WebDriverAgent.xcodeproj -showBuildSettings -configuration Debug | grep TARGET_BUILD | awk '{print $3}' | tr -d "\n")
#echo BUILD_PATH="$BPATH"
echo "$(abspath $BPATH/..)"
