package main

import (
)

func proc_device_trigger( o ProcOptions ) {
    o.procName = "device_trigger"
    o.binary = o.config.BinPaths.DeviceTrigger
    proc_generic( o )
}