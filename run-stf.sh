#!/bin/sh

IFACE=$1
export SUPPORT_ROOT=`jq .support_root config.json -j`
export MYSTF_ROOT=`jq .stf_root config.json -j`
export STF_URI=`jq .stf_ip config.json -j` # "192.168.255.1"
export STF_URI_DNS=`jq .stf_hostname config.json -j`
export OPENSTF_HOME="$MYSTF_ROOT"

STF_ROOT=`jq .stf_root config.json -j`
if [ ! -f "${STF_ROOT}/package.json" ]; then
  echo "STF folder ${STF_ROOT} does not exist!"
  exit 1
fi
# Potentially also the folder could contain a .xctestrun file instead
# TODO: Write correct code to check for that
#if [ ! -d "${WDA_ROOT}/WebDriverAgent.xcodeproj" ]; then
#  echo "WebDriverAgent folder ${WDA_ROOT} does not exist!"
#  exit 1
#fi

echo "STF ROOT   : ${MYSTF_ROOT}"
echo "STF_URI    : ${STF_URI}"
echo "STF_URI_DNS: ${STF_URI_DNS}"

export MIN_PORT=7400
export MAX_PORT=7700
export FQDN_OR_IP=`ifconfig $IFACE | grep inet | cut -d\  -f2`
export STF_SERVER_IP="$STF_URI"
export STF_CLIENT_IP="$FQDN_OR_IP"
export PROVIDER_IDENT=`hostname | tr -d "\n"`
cd $MYSTF_ROOT

node --inspect=127.0.0.1:9230 runmod.js provider \
	--name "macmini/${PROVIDER_IDENT}" \
	--connect-sub tcp://${STF_URI}:7250 \
	--connect-push tcp://${STF_URI}:7270 \
	--storage-url https://${STF_URI_DNS}/ \
	--public-ip ${FQDN_OR_IP} \
	--min-port=${MIN_PORT} \
	--max-port=${MAX_PORT} \
	--heartbeat-interval 10000 \
	--no-cleanup
