package main

import (
    "fmt"
    "strconv"
    log "github.com/sirupsen/logrus"
)

func proc_ios_video_stream( o ProcOptions, tunName string ) {
    devd := o.devd.dup()
    udid := devd.uuid
    port := o.config.MirrorFeedPort
    
    nanoIn := o.config.DecodeInPort
    nanoOut := o.config.DecodeOutPort 
    
    inSpec := fmt.Sprintf("tcp://127.0.0.1:%d", nanoIn)
    outSpec := fmt.Sprintf("tcp://127.0.0.1:%d", nanoOut)
    
    o.binary = o.config.BinPaths.IosVideoStream
    o.startFields = log.Fields {
        "tunName": tunName,
        "pushSpec": outSpec,
        "pullSpec": inSpec,
        "port": port,
    }
    o.procName = "ios_video_stream"
    o.args = []string {
        "-stream",
        "--port", strconv.Itoa( port ),
        "-udid", udid,
        "-interface", tunName,
        "-pushSpec", outSpec,
        "-pullSpec", inSpec,
    }
    proc_generic( o )
}