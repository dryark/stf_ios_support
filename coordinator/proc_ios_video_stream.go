package main

import (
    "fmt"
    "strconv"
    log "github.com/sirupsen/logrus"
)

func restart_ios_video_stream( devd *RunningDev ) {
    restart_proc_generic( devd, "ios_video_stream" )
}

func proc_ios_video_stream( o ProcOptions, tunName string, frameInIp string ) {
    devd := o.devd.dup()
    udid := devd.uuid
    port := o.config.MirrorFeedPort
    
    nanoIn := o.config.DecodeInPort
    
    inSpec := fmt.Sprintf("tcp://%s:%d", frameInIp, nanoIn)
    
    coordinator := fmt.Sprintf( "127.0.0.1:%d", o.config.Network.Coordinator )
    
    o.binary = o.config.BinPaths.IosVideoStream
    o.startFields = log.Fields {
        "tunName": tunName,
        "pullSpec": inSpec,
        "port": port,
    }
    o.procName = "ios_video_stream"
    o.args = []string {
        "-stream",
        "-port", strconv.Itoa( port ),
        "-udid", udid,
        "-interface", tunName,
        "-pullSpec", inSpec,
        "-coordinator", coordinator,
    }
    secure := o.config.FrameServer.Secure
    if secure {
        cert := o.config.FrameServer.Cert
        key := o.config.FrameServer.Key
        o.args = append( o.args,
            "--secure",
            "--cert", cert,
            "--key", key,
        )
    }
    proc_generic( o )
}