package main

import (
  "bufio"
  "os/exec"
  "strconv"
  log "github.com/sirupsen/logrus"
)

func proc_mirrorfeed( config *Config, tunName string, devd *RunningDev, lineLog *log.Entry ) {
    plog := log.WithFields( log.Fields{
        "proc": "mirrorfeed",
        "uuid": devd.uuid,
    } )
    lineLog = lineLog.WithFields( log.Fields{
        "proc": "mirrorfeed",
        "uuid": devd.uuid,
    } )

    mirrorPort := strconv.Itoa( config.MirrorFeedPort )
    mirrorFeedBin := config.MirrorFeedBin
    pipeName := devd.pipe

    if devd.shuttingDown {
        return
    }
    
    backoff := Backoff{}
    devd.mirrorBackoff = &backoff
    
    go func() {
        for {
            plog.WithFields( log.Fields{
                "type":           "proc_start",
                "mirrorfeed_bin": mirrorFeedBin,
                "pipe":           pipeName,
                "port":           mirrorPort,
                "tunName":        tunName,
            } ).Info("Starting: mirrorfeed")

            cmd := exec.Command( mirrorFeedBin, mirrorPort, pipeName, tunName, devd.uuid )

            outputPipe, _ := cmd.StdoutPipe()
            //cmd.Stderr = os.Stderr
            errPipe, _ := cmd.StderrPipe()

            backoff.markStart()
            
            err := cmd.Start()
            if err != nil {
                plog.WithFields( log.Fields{
                    "type": "proc_err",
                    "error": err,
                } ).Error("Error starting mirrorfeed")

                devd.mirror = nil
            } else {
                devd.mirror = cmd.Process
            }

            go func() {
                scanner := bufio.NewScanner( errPipe )
                for scanner.Scan() {
                    line := scanner.Text()
                    lineLog.WithFields( log.Fields{ "line": line, "iserr": true } ).Info("")
                }
            } ()
            scanner := bufio.NewScanner( outputPipe )
            for scanner.Scan() {
                line := scanner.Text()
                lineLog.WithFields( log.Fields{ "line": line } ).Info("")
            }
            
            cmd.Wait()
            backoff.markEnd()

            devd.mirror = nil

            plog.WithFields( log.Fields{  "type": "proc_end" } ).Warn("Ended: mirrorfeed")

            devd.lock.Lock()
            exit := devd.shuttingDown
            devd.lock.Unlock()
            if exit { break }
            
            backoff.wait()
        }
    }()
}