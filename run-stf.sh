#!/bin/sh

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"
source $DIR"/config.sh"
echo "OpenSTF HOME: ${OPENSTF_HOME}"

# function, check element in array
in_array() {
  local array=${1}[@]
  local needle=${2}
  for i in ${!array}; do
    if [[ ${i} == ${needle} ]]; then
        return 0
    fi
  done
  return 1
}


# Check and install dependencies
echo "Check dependencies:"
installed=( $(brew list) )
not_installed=()
i=0
for dependency in ${STF_DEPENDENCIES[@]}; do
  if ! in_array installed ${dependency}; then
    not_installed[$i]=${dependency}
    i=$(($i+1))
  fi
done
if [ ${#not_installed[@]} == 0 ]; then
  echo "All dependencies installed"
else
  for lib in ${not_installed[@]}; do
    echo "Install ${lib}"
    brew install ${lib}
  done
fi

echo "Start stf:"
cd ${OPENSTF_HOME}/repos/stf/
echo "npm install/link"
#npm install
#npm link

export STF_PROVIDER_NAME="mac1"

export MIN_PORT=7400
export MAX_PORT=7700
export FQDN_OR_IP=`ifconfig utun1 | grep inet | cut -d\  -f2`
export STF_URI="192.168.255.1"
export STF_URI_DNS="[ your stf hostname ]"
export STF_SERVER_IP="192.168.255.1"
export STF_CLIENT_IP="$FQDN_OR_IP"
stf provider \
	--name "${STF_PROVIDER_NAME}/${STF_PROVIDER_NAME}" \
	--connect-sub tcp://${STF_URI}:7250 \
	--connect-push tcp://${STF_URI}:7270 \
	--storage-url https://${STF_URI_DNS}/ \
	--public-ip ${FQDN_OR_IP} \
	--min-port=${MIN_PORT} \
	--max-port=${MAX_PORT} \
	--heartbeat-interval 10000 \
	--no-cleanup \
	--screen-ws-url-pattern "wss://${STF_URI_DNS}/d/${STF_PROVIDER_NAME}/<%= serial %>/<%= publicPort %>/"
