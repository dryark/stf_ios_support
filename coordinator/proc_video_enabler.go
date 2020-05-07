package main

import (
)

func proc_video_enabler( o ProcOptions ) {
    o.procName = "video_enabler"
    o.binary = o.config.BinPaths.VideoEnabler
    proc_generic( o )
}