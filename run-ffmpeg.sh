#!/bin/bash
BIN="$1"
PIPE="$2"
DEVNAME="$3"
shift 3
$BIN -f avfoundation -i "$DEVNAME" $* > $PIPE
