#!/bin/bash

CONFNAME="$1"
VPERR=""

# Junk below mostly copied from Tunnelblick client.2.up.tunnelblick.sh
TBCONFIG="/Users/davidh/Library/Application Support/Tunnelblick/Configurations/$1.tblk/Contents/Resources/config.ovpn"
if [ -f "$TBCONFIG" ]; then
    TBALTPREFIX="/Library/Application Support/Tunnelblick/Users/"
    TBALTPREFIXLEN="${#TBALTPREFIX}"
    TBCONFIGSTART="${TBCONFIG:0:$TBALTPREFIXLEN}"
    if [ "$TBCONFIGSTART" = "$TBALTPREFIX" ] ; then
        TBBASE="${TBCONFIG:$TBALTPREFIXLEN}"
        TBSUFFIX="${TBBASE#*/}"
        TBUSERNAME="${TBBASE%%/*}"
        TBCONFIG="/Users/$TBUSERNAME/Library/Application Support/Tunnelblick/Configurations/$TBSUFFIX"
    fi
    CONFIG_PATH_DASHES_SLASHES="$(echo "${TBCONFIG}" | sed -e 's/-/--/g' | sed -e 's/\//-S/g')"
    
    # Determine IP adddress and Tunnel name of most recent connection
    LF=$(find "/Library/Application Support/Tunnelblick/Logs/" -type f -iname "${CONFIG_PATH_DASHES_SLASHES}*.openvpn.log")
    if [ -f "$LF" ]; then
        LINE=$(cat "$LF" | grep -E "ifconfig utun[0-9]+ [0-9]+" | tail -1)
        if [ "$LINE" == "" ]; then
            VPERR="Config '$CONFNAME' does not appear to have ever connected"
        else
            IPADDR=$(echo $LINE| cut -d \  -f 5)
            TUN=$(echo $LINE| cut -d \  -f 4)
        fi
    else
        VPERR="Config '$CONFNAME' does not have any log"
    fi
else
    VPERR="Config '$CONFNAME' does not appear to exist in Tunnelblick"
fi

echo "{"
echo "  \"err\": \"$VPERR\","
echo "  \"ipAddr\": \"$IPADDR\","
echo "  \"tunName\": \"$TUN\""
echo "}"