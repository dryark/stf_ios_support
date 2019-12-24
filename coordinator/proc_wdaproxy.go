package main

import (
  "bufio"
  "fmt"
  "os/exec"
  "strconv"
  "strings"
  "time"
  log "github.com/sirupsen/logrus"
)

func proc_wdaproxy(
        config     *Config,
        devd       *RunningDev,
        devEvent   *DevEvent,
        uuid       string,
        devName    string,
        pubEventCh chan<- PubEvent,
        lineLog    *log.Entry,
        iosVersion string ) {
    plog := log.WithFields( log.Fields{
        "proc": "wdaproxy",
        "uuid": devd.uuid,
    } )
    lineLog = lineLog.WithFields( log.Fields{
        "proc": "wdaproxy",
        "uuid": devd.uuid,
    } )

    // start wdaproxy
    wdaPort := config.WDAProxyPort

    if devd.shuttingDown {
        return
    }
    
    backoff := Backoff{}
    devd.proxyBackoff = &backoff
    
    go func() {
        for {
            ops := []string{
              "-p", strconv.Itoa( wdaPort ),
              "-q", strconv.Itoa( wdaPort ),
              "-d",
              "-W", ".",
              "-u", uuid,
              fmt.Sprintf("--iosversion=%s", iosVersion),
            }

            plog.WithFields( log.Fields{
              "type":       "proc_start",
              "port":       wdaPort,
              "wda_folder": config.WDARoot,
              "ops":        ops,
            } ).Info("Starting wdaproxy")

            cmd := exec.Command( "../../bin/wdaproxy", ops... )

            cmd.Dir = config.WDARoot

            outputPipe, _ := cmd.StdoutPipe()
            errPipe, _ := cmd.StderrPipe()
            
            backoff.markStart()
            err := cmd.Start()
            if err != nil {
                plog.WithFields( log.Fields{
                    "type": "proc_err",
                    "error": err,
                } ).Error("Error starting wdaproxy")

                devd.proxy = nil
            } else {
                devd.proxy = cmd.Process
            }

            time.Sleep( time.Second * 3 )

            // Everything is started; notify stf via zmq published event
            pubEvent := PubEvent{}
            pubEvent.action  = devEvent.action
            pubEvent.uuid    = devEvent.uuid
            pubEvent.name    = devName
            pubEvent.wdaPort = config.WDAProxyPort
            pubEvent.vidPort = config.MirrorFeedPort
            pubEventCh <- pubEvent

            stopChannel := coro_heartbeat( devEvent, pubEventCh )

            go func() {
                scanner := bufio.NewScanner( outputPipe )
                for scanner.Scan() {
                    line := scanner.Text()

                    if strings.Contains( line, "is implemented in both" ) {
                    } else if strings.Contains( line, "Couldn't write value" ) {
                    } else if strings.Contains( line, "GET /status " ) {
                    } else {
                        lineLog.WithFields( log.Fields{ "line": line } ).Info("")
                    }
                }
            } ()
            scanner := bufio.NewScanner( errPipe )
            for scanner.Scan() {
                line := scanner.Text()

                if strings.Contains( line, "[WDA] successfully started" ) {
                    plog.WithFields( log.Fields{ "type": "wda_started" } ).Info("WDA started")
                }

                lineLog.WithFields( log.Fields{ "line": line, "iserr": true } ).Info("")
            }
            
            stopChannel<- true
            cmd.Wait()
            
            backoff.markEnd()
            
            plog.WithFields( log.Fields{ "type": "proc_end" } ).Warn("Ended: wdaproxy")

            devd.lock.Lock()
            exit := devd.shuttingDown
            devd.lock.Unlock()
            if exit { break }
            
            backoff.wait()
        }
    }()
}