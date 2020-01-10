package main

import (
  "bufio"
  "os/exec"
  "strconv"
  log "github.com/sirupsen/logrus"
)

func proc_vnc_proxy( config *Config, devd *RunningDev, lineLog *log.Entry ) {
    plog := log.WithFields( log.Fields{
        "proc": "vnc_proxy",
        "uuid": devd.uuid,
    } )
    lineLog = lineLog.WithFields( log.Fields{
        "proc": "vnc_proxy",
        "uuid": devd.uuid,
    } )

    vncPort := config.VncPort
    iproxyBin := config.IProxyBin //"/usr/local/bin/iproxy"
    
    if devd.shuttingDown {
        return
    }
    
    backoff := Backoff{}
    devd.iproxyBackoff = &backoff
    
    go func() {
        for {
            plog.WithFields( log.Fields{
                "type":       "proc_start",
                "iproxy_bin": iproxyBin,
                "vncPort":    vncPort,
            } ).Info("Process start - vnc proxy")

            cmd := exec.Command( "/usr/local/bin/iproxy", strconv.Itoa( vncPort ), "5900" )

            outputPipe, _ := cmd.StdoutPipe()
            errPipe, _ := cmd.StderrPipe()

            backoff.markStart()
            
            err := cmd.Start()
            if err != nil {
                plog.WithFields( log.Fields{
                    "type": "proc_err",
                    "error": err,
                } ).Error("Error starting vnc proxy")

                devd.iproxy = nil
            } else {
                devd.iproxy = cmd.Process
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

            devd.iproxy = nil

            plog.WithFields( log.Fields{  "type": "proc_end" } ).Warn("Process end - vnc proxy")

            devd.lock.Lock()
            exit := devd.shuttingDown
            devd.lock.Unlock()
            if exit { break }
            
            backoff.wait()
        }
    }()
}