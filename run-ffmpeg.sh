#!/bin/bash
BIN="$1"
PIPE="$2"
DEVNAME="$3"
shift 3
echo Running $BIN $* into $PIPE
$BIN -f avfoundation -i "$DEVNAME" $* > $PIPE
