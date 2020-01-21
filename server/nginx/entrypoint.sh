#!/bin/sh

NAMESERVER=$(awk '/nameserver/{print $2}' /etc/resolv.conf | tr '\\n' ' ')
RESOLVER_CONFIG="/etc/nginx/conf.d/resolver.conf"
echo Got nameserver $NAMESERVER from resolv.conf
echo Writing include file at $RESOLVER_CONFIG
echo "resolver $NAMESERVER;" > $RESOLVER_CONFIG
nginx -g 'daemon off;'
