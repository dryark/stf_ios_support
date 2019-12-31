package main

import (
  "fmt"
  "os"
  "strconv"
)

func start_proc_wdaproxy(
        config     *Config,
        devd       *RunningDev,
        uuid       string,
        iosVersion string ) {

    if devd.shuttingDown {
        return
    }
    
    arguments := []string {
        config.WDAWrapperBin,
        "-port", strconv.Itoa(config.WDAProxyPort),
        "-uuid", uuid,
        "-iosVersion", iosVersion,
        "-wdaRoot", config.WDARoot,
    }
    
    label := fmt.Sprintf("com.tmobile.coordinator.wdawrapper_%s", uuid )
    wd, _ := os.Getwd()
    keepalive := true
    stfLauncher := NewLauncher( label, arguments, keepalive, wd )
    stfLauncher.stdout = config.WDAWrapperStdout
    stfLauncher.stderr = config.WDAWrapperStderr
    stfLauncher.load()
    
    devd.wdaWrapper = stfLauncher
}

func stop_proc_wdaproxy( devd *RunningDev ) {
    devd.wdaWrapper.unload()
}