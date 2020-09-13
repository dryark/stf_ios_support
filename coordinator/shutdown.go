package main

import (
  "os"
  "os/signal"
  //"strings"
  "syscall"
  "time"
  log "github.com/sirupsen/logrus"
  ps "github.com/jviney/go-proc"
)

func cleanup_procs(config *Config) {
    plog := log.WithFields( log.Fields{
        "type": "proc_cleanup_kill",
    } )

    procMap := map[string]string {
        "ios_video_stream": config.BinPaths.IosVideoStream,
        "device_trigger": config.BinPaths.DeviceTrigger,
        "h264_to_jpeg": config.BinPaths.H264ToJpeg,
        "wdaproxy": "../wdaproxy",
    }
    
    // Cleanup hanging processes if any
    procs := ps.GetAllProcessesInfo()
    for _, proc := range procs {
        cmd := proc.CommandLine
        //cmdFlat := strings.Join( cmd, " " )
        
        for k,v := range procMap {
            if cmd[0] == v {
                plog.WithFields( log.Fields{ "proc": k } ).Info("Leftover " + k + " - Sending SIGTERM")
                syscall.Kill( proc.Pid, syscall.SIGTERM )
            }
        }
        
        // node --inspect=[ip]:[port] runmod.js device-ios
        if cmd[0] == "/usr/local/opt/node@12/bin/node" && cmd[3] == "device-ios" {
            plog.WithFields( log.Fields{
                "proc": "device-ios",
            } ).Debug("Leftover Proc - Sending SIGTERM")

            syscall.Kill( proc.Pid, syscall.SIGTERM )
        }

        // node --inspect=[ip]:[port] runmod.js provider
        if cmd[0] == "/usr/local/opt/node@12/bin/node" && cmd[3] == "provider" {
            plog.WithFields( log.Fields{
                "proc": "stf_provider",
            } ).Debug("Leftover Proc - Sending SIGTERM")

            syscall.Kill( proc.Pid, syscall.SIGTERM )
        }
    }
}

func closeAllRunningDevs( runningDevs map [string] *RunningDev ) {
    for _, devd := range runningDevs {
        closeRunningDev( devd, nil )
    }
}

func closeRunningDev( devd *RunningDev, portMap *PortMap ) {
    devd.lock.Lock()
    devd.shuttingDown = true
    devd.lock.Unlock()
    
    if portMap != nil {
        free_ports( devd.wdaPort, devd.vidPort, devd.devIosPort, devd.vncPort, devd.usbmuxdPort, portMap )
    }

    plog := log.WithFields( log.Fields{
        "type": "proc_cleanup_kill",
        "uuid": censor_uuid( devd.uuid ),
    } )

    plog.Info("Closing running dev")

    for k,v := range( devd.process ) {
        plog.WithFields( log.Fields{ "proc": k } ).Debug("Killing "+k)
        if v != nil { v.Kill() }
    }
}

func closeBaseProgs( baseProgs *BaseProgs ) {
    baseProgs.shuttingDown = true
    vpn_shutdown( baseProgs )
    
    plog := log.WithFields( log.Fields{ "type": "proc_cleanup_kill" } )

    for k,v := range( baseProgs.process ) {
        plog.WithFields( log.Fields{ "proc": k } ).Debug("Killing "+k)
        v.Kill()
    }
}

func coro_sigterm( runningDevs map [string] *RunningDev, baseProgs *BaseProgs, config *Config ) {
    c := make(chan os.Signal, 2)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    go func() {
        <- c
        log.WithFields( log.Fields{
            "type": "sigterm",
            "state": "begun",
        } ).Info("Shutdown started")

        // This triggers zmq to stop receiving
        // We don't actually wait after this to ensure it has finished cleanly... oh well :)
        gStop = true
        
        closeAllRunningDevs( runningDevs )
        closeBaseProgs( baseProgs )
        
        time.Sleep( time.Millisecond * 1000 )
        cleanup_procs( config )

        log.WithFields( log.Fields{
            "type": "sigterm",
            "state": "done",
        } ).Info("Shutdown finished")

        os.Exit(0)
    }()
}