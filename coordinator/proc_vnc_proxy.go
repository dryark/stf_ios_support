package main

import (
  "strconv"
  log "github.com/sirupsen/logrus"
)

func proc_vnc_proxy( o ProcOptions ) {
    o.procName = "vnc_proxy"
    
    vncPort := o.config.VncPort
    o.binary      = o.config.BinPaths.Iproxy
    o.startFields = log.Fields {
        "vncPort": vncPort,
    }
    o.args = []string {
        strconv.Itoa( vncPort ), "5900",
    }
    proc_generic( o )
}