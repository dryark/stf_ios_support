package main

import (
    "fmt"
    "strconv"
    "strings"
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
    
    width := o.config.FrameServer.Width
    height := o.config.FrameServer.Height
    
    if width != 0 {
        o.args = append( o.args, "--dw", strconv.Itoa( width ) )
    }
    if height != 0 {
        o.args = append( o.args, "--dh", strconv.Itoa( height ) )
    }
    
    o.stdoutHandler = func( line string, plog *log.Entry ) (bool) {
        if strings.Contains( line, "Iframe - size:" ) {
            return false
        }
        return true
    }
    
    proc_generic( o )
}