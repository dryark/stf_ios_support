package main

import (
)

func proc_device_trigger( o ProcOptions ) {
    o.procName = "device_trigger"
    
    conf := o.config
    if conf.DeviceDetector == "api" {
        o.binary = o.config.BinPaths.IosDeploy
        o.args = []string{
            "-d",
            "-n", "test",
            "-t", "0",
        }
    } else {
        o.binary = o.config.BinPaths.DeviceTrigger
    }
    
    proc_generic( o )
}