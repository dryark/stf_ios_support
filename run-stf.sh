#!/bin/sh

IFACE=$1
STF_IP=`jq .stf_ip config.json -j`
STF_HOSTNAME=`jq .stf_hostname config.json -j`
PUBLIC_IP=`ifconfig $IFACE | grep inet | cut -d\  -f2`
PROVIDER_IDENT=`hostname | tr -d "\n"`

cd repos/stf
node --inspect=127.0.0.1:9230 runmod.js provider \
	--name "macmini/${PROVIDER_IDENT}" \
	--connect-sub tcp://${STF_IP}:7250 \
	--connect-push tcp://${STF_IP}:7270 \
	--storage-url https://${STF_HOSTNAME}/ \
	--public-ip ${PUBLIC_IP} \
	--min-port=7400 \
	--max-port=7700 \
	--heartbeat-interval 10000 \
	--server-ip=${STF_IP} \
	--no-cleanup
