package main

import (
  "os"
  "os/signal"
  "strings"
  "syscall"
  "time"
  log "github.com/sirupsen/logrus"
  ps "github.com/jviney/go-proc"
)

func cleanup_procs(config *Config) {
    plog := log.WithFields( log.Fields{
        "type": "proc_cleanup_kill",
    } )

    // Cleanup hanging processes if any
    procs := ps.GetAllProcessesInfo()
    for _, proc := range procs {
        cmd := proc.CommandLine
        cmdFlat := strings.Join( cmd, " " )
        if cmd[0] == "bin/ffmpeg" {
            plog.WithFields( log.Fields{
                "proc": "ffmpeg",
            } ).Debug("Leftover FFmpeg - Sending SIGTERM")

            syscall.Kill( proc.Pid, syscall.SIGTERM )
        }
        if cmd[0] == "bin/mirrorfeed" {
            plog.WithFields( log.Fields{
                "proc": "mirrorfeed",
            } ).Debug("Leftover Mirrorfeed - Sending SIGTERM")

            syscall.Kill( proc.Pid, syscall.SIGTERM )
        }
        if cmdFlat == config.BinPaths.VideoEnabler {
            plog.WithFields( log.Fields{
                "proc": "video_enabler",
            } ).Debug("Leftover Proc - Sending SIGTERM")

            syscall.Kill( proc.Pid, syscall.SIGTERM )
        }
        if cmdFlat == config.BinPaths.DeviceTrigger {
            plog.WithFields( log.Fields{
                "proc": "device_trigger",
            } ).Debug("Leftover Proc - Sending SIGTERM")

            syscall.Kill( proc.Pid, syscall.SIGTERM )
        }

        // node --inspect=[ip]:[port] runmod.js device-ios
        if cmd[0] == "/usr/local/opt/node@8/bin/node" && cmd[3] == "device-ios" {
            plog.WithFields( log.Fields{
                "proc": "device-ios",
            } ).Debug("Leftover Proc - Sending SIGTERM")

            syscall.Kill( proc.Pid, syscall.SIGTERM )
        }

        // node --inspect=[ip]:[port] runmod.js provider
        if cmd[0] == "/usr/local/opt/node@8/bin/node" && cmd[3] == "provider" {
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
    
    stop_proc_wdaproxy( devd )

    if portMap != nil {
        free_ports( devd.wdaPort, devd.vidPort, devd.devIosPort, devd.vncPort, portMap )
    }

    plog := log.WithFields( log.Fields{
        "type": "proc_cleanup_kill",
        "uuid": devd.uuid,
    } )

    plog.Info("Closing running dev")

    /*if devd.proxy != nil {
        plog.WithFields( log.Fields{ "proc": "wdaproxy" } ).Debug("Killing wdaproxy")
        devd.proxy.Kill()
    }*/
    if devd.ff != nil {
        plog.WithFields( log.Fields{ "proc": "ffmpeg" } ).Debug("Killing ffmpeg")
        devd.ff.Kill()
    }
    if devd.mirror != nil {
        plog.WithFields( log.Fields{ "proc": "mirrorfeed" } ).Debug("Killing mirrorfeed")
        devd.mirror.Kill()
    }
    if devd.device != nil {
        plog.WithFields( log.Fields{ "proc": "device_ios_unit" } ).Debug("Killing device_ios_unit")
        devd.device.Kill()
    }
}

func closeBaseProgs( baseProgs *BaseProgs ) {
    baseProgs.shuttingDown = true
    vpn_shutdown( baseProgs )
    
    plog := log.WithFields( log.Fields{
        "type": "proc_cleanup_kill",
    } )

    if baseProgs.trigger != nil {
        plog.WithFields( log.Fields{ "proc": "device_trigger" } ).Debug("Killing device_trigger")
        baseProgs.trigger.Kill()
    }
    if baseProgs.vidEnabler != nil {
        plog.WithFields( log.Fields{ "proc": "video_enabler" } ).Debug("Killing video_enabler")
        baseProgs.vidEnabler.Kill()
    }
    if baseProgs.stf != nil {
        plog.WithFields( log.Fields{ "proc": "stf_provider" } ).Debug("Killing stf_provider")
        baseProgs.stf.Kill()
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

        closeAllRunningDevs( runningDevs )
        closeBaseProgs( baseProgs )
        
        // This triggers zmq to stop receiving
        // We don't actually wait after this to ensure it has finished cleanly... oh well :)
        gStop = true

        time.Sleep( time.Millisecond * 1000 )
        cleanup_procs( config )

        log.WithFields( log.Fields{
            "type": "sigterm",
            "state": "done",
        } ).Info("Shutdown finished")

        os.Exit(0)
    }()
}