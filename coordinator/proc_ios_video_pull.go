package main

import (
    "fmt"
    //"strconv"
    "strings"
    log "github.com/sirupsen/logrus"
)

func proc_ios_video_pull( o ProcOptions ) {
    devd := o.devd.dup()
    udid := devd.uuid
    
    nanoOut := o.config.DecodeOutPort 
    
    outSpec := fmt.Sprintf("tcp://127.0.0.1:%d", nanoOut)
    
    o.binary = o.config.BinPaths.IosVideoPull
    o.startFields = log.Fields {
        "pushSpec": outSpec,
    }
    o.procName = "ios_video_pull"
    o.args = []string {
        "-pull",
        "-udid", udid,
        "-pushSpec", outSpec,
    }
    o.stdoutHandler = func( line string, plog *log.Entry ) (bool) {
        if strings.Contains( line, "error: libusb: interrupted" ) {
            return false
        }
        return true
    }
    proc_generic( o )
}