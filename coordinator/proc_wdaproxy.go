package main

import (
  "fmt"
  "os"
  "strconv"
)

func start_proc_wdaproxy( o ProcOptions, uuid string, iosVersion string ) {
    if o.devd.shuttingDown {
        return
    }
    
    arguments := []string {
        o.config.BinPaths.WdaWrapper,
        "-port", strconv.Itoa(o.config.WDAProxyPort),
        "-uuid", uuid,
        "-iosVersion", iosVersion,
        "-wdaRoot", o.config.WdaFolder,
    }
    
    label := fmt.Sprintf("com.tmobile.coordinator.wdawrapper_%s", uuid )
    wd, _ := os.Getwd()
    keepalive := true
    asRoot := false
    stfLauncher := NewLauncher( label, arguments, keepalive, wd, asRoot )
    stfLauncher.stdout = o.config.Log.WDAWrapperStdout
    stfLauncher.stderr = o.config.Log.WDAWrapperStderr
    stfLauncher.load()
    
    o.devd.wdaWrapper = stfLauncher
}

func stop_proc_wdaproxy( devd *RunningDev ) {
    if devd.wdaWrapper != nil {
        devd.wdaWrapper.unload()
        devd.wdaWrapper = nil
    }
}