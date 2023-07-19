package main

import (
	"fmt"
    "os"
    "os/signal"
    //"strings"
    "syscall"
    "time"
    log "github.com/sirupsen/logrus"
    //ps "github.com/jviney/go-proc"
    si "github.com/elastic/go-sysinfo"
)

func cleanup_procs(config *Config) {
    plog := log.WithFields( log.Fields{
        "type": "proc_cleanup",
    } )

    procMap := map[string]string {
        "ios_video_stream": config.BinPaths.IosVideoStream,
        "device_trigger":   config.BinPaths.DeviceTrigger,
        "h264_to_jpeg":     config.BinPaths.H264ToJpeg,
        "wdaproxy":         "../wdaproxy",
        "ivf":              config.BinPaths.IVF,
        "ios-deploy":       config.BinPaths.IosDeploy,
    }

	// Cleanup hanging processes if any
    procs, listErr := si.Processes()
    if listErr != nil {
    	fmt.Printf( "listErr:%s\n", listErr )
    	os.Exit(1)
    }

    var hangingPids []int

    for _, proc := range procs {
    	info, infoErr := proc.Info()
    	if infoErr != nil {
    	    //fmt.Printf( "infoErr:%s\n", infoErr )
    	    continue
    	}

        cmd := info.Args
        //cmdFlat := strings.Join( cmd, " " )

        for k,v := range procMap {
            if cmd[0] == v {
                pid := proc.PID()
                plog.WithFields( log.Fields{
                    "proc": k,
                    "pid":  pid,
                } ).Warn("Leftover " + k + " - Sending SIGTERM")

                syscall.Kill( pid, syscall.SIGTERM )
                hangingPids = append( hangingPids, pid )
            }
        }

        /*if strings.Contains( cmdFlat, "node" ) {
        	log.WithFields( log.Fields{
                "cmdLine": cmdFlat,
            } ).Info("Leftover Node proc")
        }*/

        // node --inspect=[ip]:[port] runmod.js device-ios
        if cmd[0] == config.BinPaths.Node && cmd[3] == "device-ios" {
            pid := proc.PID()

            plog.WithFields( log.Fields{
                "proc": "device-ios",
                "pid":  pid,
            } ).Warn("Leftover Proc - Sending SIGTERM")

            syscall.Kill( pid, syscall.SIGTERM )
            hangingPids = append( hangingPids, pid )
        }

        // node --inspect=[ip]:[port] runmod.js provider
        if cmd[0] == config.BinPaths.Node && cmd[3] == "provider" {
            pid := proc.PID()

            plog.WithFields( log.Fields{
                "proc": "stf_provider",
                "pid":  pid,
            } ).Warn("Leftover Proc - Sending SIGTERM")

            syscall.Kill( pid, syscall.SIGTERM )
            hangingPids = append( hangingPids, pid )
        }
    }

    if len( hangingPids ) > 0 {
        // Give the processes half a second to shudown cleanly
        time.Sleep( time.Millisecond * 500 )

        // Send kill to processes still around
        for _, pid := range( hangingPids ) {
            proc, _ := si.Process( pid )
            if proc != nil {
                info, infoErr := proc.Info()
                arg0 := "unknown"
                if infoErr == nil {
                    args := info.Args
                    arg0 = args[0]
                } else {
                    // If the process vanished before here; it errors out fetching info
                    continue
                }

                plog.WithFields( log.Fields{
                    "arg0": arg0,
                } ).Warn("Leftover Proc - Sending SIGKILL")
                syscall.Kill( pid, syscall.SIGKILL )
            }
        }

        // Spend up to 500 ms waiting for killed processes to vanish
        i := 0
        for {
            i = i + 1
            time.Sleep( time.Millisecond * 100 )
            allGone := 1
            for _, pid := range( hangingPids ) {
                proc, _ := si.Process( pid )
                if proc != nil {
                    _, infoErr := proc.Info()
                    if infoErr != nil {
                        continue
                    }
                    allGone = 0
                }
            }
            if allGone == 1 && i > 5 {
                break
            }
        }

        // Write out error messages for processes that could not be killed
        for _, pid := range( hangingPids ) {
                proc, _ := si.Process( pid )
                if proc != nil {
                    info, infoErr := proc.Info()
                    arg0 := "unknown"
                    if infoErr != nil {
                        continue
                    }
                    args := info.Args
                    arg0 = args[0]

                    plog.WithFields( log.Fields{
                        "arg0": arg0,
                    } ).Error("Kill attempted and failed")
                }
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
            "type":  "sigterm",
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
            "type":  "sigterm",
            "state": "done",
        } ).Info("Shutdown finished")

        os.Exit(0)
    }()
}
