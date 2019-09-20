#!/bin/sh
# This script config params, check dependency command tools installed, and check all project exist
# Please config DEVELOPMENT_TEAM WDA_BUNDLE_ID according to your xcode setting

# unique bundle ID of WebDriverAgent
WDA_BUNDLE_ID="com.facebook.WebDriverAgentLib"

# directory of OPENSTF project
WDA_ROOT=`jq .wda_root config.json -j` # "/Users/davidh/git/openstf-ios-extended/"
STF_ROOT=`jq .stf_root config.json -j`

# your development team id of xcode
DEVELOPMENT_TEAM=`jq .xcode_dev_team_id config.json -j` # "[your devteam id]"

if [ ! -f "${STF_ROOT}/package.json" ]; then
  echo "STF folder ${STF_ROOT} does not exist!"
  exit 1
fi
if [ ! -d "${WDA_ROOT}/WebDriverAgent.xcodeproj" ]; then
  echo "WebDriverAgent folder ${WDA_ROOT} does not exist!"
  exit 1
fi
