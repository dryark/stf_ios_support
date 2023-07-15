package main

import (
  "fmt"
  "os"
  "strings"
  log "github.com/sirupsen/logrus"
)

func proc_stf_provider( o ProcOptions, curIP string ) {
    o.binary = o.config.BinPaths.IosVideoStream

    serverHostname := o.config.Stf.HostName
    clientHostname, _ := os.Hostname()
    serverIP := o.config.Stf.Ip

    location := fmt.Sprintf("macmini/%s", clientHostname)
    if o.config.Stf.Location != "" {
        location = o.config.Stf.Location
    }

    o.startFields = log.Fields {
        "client_ip":       curIP,
        "server_ip":       serverIP,
        "client_hostname": clientHostname,
        "server_hostname": serverHostname,
        "location":        location,
    }
    o.binary = "/usr/local/opt/node@19/bin/node"
    o.args = []string {
        "--inspect=127.0.0.1:9230",
        "runmod.js"      , "provider",
        "--name"         , location,
        "--connect-sub"  , fmt.Sprintf("tcp://%s:7250", serverIP),
        "--connect-push" , fmt.Sprintf("tcp://%s:7270", serverIP),
        "--storage-url"  , fmt.Sprintf("https://%s", serverHostname),
        "--public-ip"    , curIP,
        "--min-port=7400",
        "--max-port=7700",
        "--heartbeat-interval=10000",
        "--server-ip"    , serverIP,
        "--no-cleanup",
    }
    o.procName = "stf_ios_provider"
    o.startDir = "./repos/stf-ios-provider"
    o.stdoutHandler = func( line string, plog *log.Entry  ) (bool) {
        if strings.Contains( line, " IOS Heartbeat:" ) {
            return false
        }
        return true
    }
    proc_generic( o )
}