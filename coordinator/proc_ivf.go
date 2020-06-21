package main

import (
    "fmt"
    //"strconv"
    //"strings"
    log "github.com/sirupsen/logrus"
)

func proc_ivf( o ProcOptions ) {
    devd := o.devd.dup()
    udid := devd.uuid
    
    nanoIn := o.config.DecodeInPort
    toStreamSpec := fmt.Sprintf("tcp://127.0.0.1:%d", nanoIn)
    
    o.binary = o.config.BinPaths.IVF
    o.startFields = log.Fields {
        "outSpec": toStreamSpec,
    }
    o.procName = "ivf"
    o.args = []string {
        "nano",
        "--udid", udid,
        "--out", toStreamSpec,
        "--frameSkip", "2",
    }
    proc_generic( o )
}