package main

import (
    "fmt"
    log "github.com/sirupsen/logrus"
)

func proc_h264_to_jpeg( o ProcOptions ) {
    devd := o.devd.dup()
    udid := devd.uuid
    
    nanoIn := o.config.DecodeInPort
    nanoOut := o.config.DecodeOutPort 
    
    inSpec := fmt.Sprintf("tcp://127.0.0.1:%d", nanoIn)
    outSpec := fmt.Sprintf("tcp://127.0.0.1:%d", nanoOut)
    
    o.binary = o.config.BinPaths.H264ToJpeg
    o.startFields = log.Fields {
        "h264SrcSpec": outSpec,
        "jpegDestSpec": inSpec,
    }
    o.procName = "h264_to_jpeg"
    o.args = []string {
        "nano",
        "--in", outSpec,
        "--out", inSpec,
        "--frameSkip", "2",
        "--cacheid", udid,
    }
    proc_generic( o )
}