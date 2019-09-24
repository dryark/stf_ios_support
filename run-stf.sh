#!/bin/sh

export SUPPORT_ROOT=`jq .support_root config.json -j`
export MYSTF_ROOT=`jq .stf_root config.json -j`
export STF_URI=`jq .stf_ip config.json -j` # "192.168.255.1"
export STF_URI_DNS=`jq .stf_hostname config.json -j`
export OPENSTF_HOME="$MYSTF_ROOT"

source "$SUPPORT_ROOT/config.sh"

echo "STF ROOT   : ${MYSTF_ROOT}"
echo "STF_URI    : ${STF_URI}"
echo "STF_URI_DNS: ${STF_URI_DNS}"

echo "Start stf:"
cd ${STF_ROOT}
#echo "npm install/link"
#npm install
#npm link

export MIN_PORT=7400
export MAX_PORT=7700
export FQDN_OR_IP=`ifconfig utun1 | grep inet | cut -d\  -f2`
export STF_SERVER_IP="$STF_URI"
export STF_CLIENT_IP="$FQDN_OR_IP"
export PROVIDER_IDENT=`hostname | tr -d "\n"`
export STF_PROVIDER_NAME="mac1"
cd $MYSTF_ROOT
#alias mystf="node --inspect=127.0.0.1:9230 runplease.js"
# \
node --inspect=127.0.0.1:9230 runplease.js provider \
	--name "macmini/${PROVIDER_IDENT}" \
	--connect-sub tcp://${STF_URI}:7250 \
	--connect-push tcp://${STF_URI}:7270 \
	--storage-url https://${STF_URI_DNS}/ \
	--public-ip ${FQDN_OR_IP} \
	--min-port=${MIN_PORT} \
	--max-port=${MAX_PORT} \
	--heartbeat-interval 10000 \
	--no-cleanup \
	--screen-ws-url-pattern "wss://${STF_URI_DNS}/d/${STF_PROVIDER_NAME}/<%= serial %>/<%= publicPort %>/"
