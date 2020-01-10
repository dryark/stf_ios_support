package main

import (
  "bufio"
  "os/exec"
  log "github.com/sirupsen/logrus"
)

func proc_device_trigger( config *Config, baseProgs *BaseProgs, lineLog *log.Entry ) {
    plog := log.WithFields( log.Fields{ "proc": "device_trigger" } )
    lineLog = lineLog.WithFields( log.Fields{ "proc": "device_trigger" } )
    
    go func() {
        plog.WithFields( log.Fields{ "type": "proc_start" } ).Info("Process start - device_trigger")

        cmd := exec.Command( config.DeviceTrigger )

        outputPipe, _ := cmd.StdoutPipe()
        errPipe, _ := cmd.StderrPipe()
        
        err := cmd.Start()
        if err != nil {
            plog.WithFields( log.Fields{
                "type": "proc_err",
                "error": err,
            } ).Error("Error starting device_trigger")

            baseProgs.trigger = nil
        } else {
            baseProgs.trigger = cmd.Process
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

        plog.WithFields( log.Fields{ "type": "proc_end" } ).Warn("Process end - device_trigger")
    }()
}