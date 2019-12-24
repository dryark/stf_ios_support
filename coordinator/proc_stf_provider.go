package main

import (
  "bufio"
  "fmt"
  "os"
  "os/exec"
  "strings"
  log "github.com/sirupsen/logrus"
)

func proc_stf_provider( baseProgs *BaseProgs, curIP string, config *Config, lineLog *log.Entry ) {
    plog := log.WithFields( log.Fields{ "proc": "stf_provider" } )
    lineLog = lineLog.WithFields( log.Fields{
        "proc": "stf_provider",
    } )
    
    backoff := Backoff{}
    
    go func() {
        for {
            serverHostname := config.STFHostname
            clientHostname, _ := os.Hostname()
            serverIP := config.STFIP

            plog.WithFields( log.Fields{
                "type":            "proc_start",
                "client_ip":       curIP,
                "server_ip":       serverIP,
                "client_hostname": clientHostname,
                "server_hostname": serverHostname,
            } ).Info("Starting: stf_provider")

            cmd := exec.Command( "/usr/local/opt/node@8/bin/node",
                "--inspect=127.0.0.1:9230",
                "runmod.js"      , "provider",
                "--name"         , fmt.Sprintf("macmini/%s", clientHostname),
                "--connect-sub"  , fmt.Sprintf("tcp://%s:7250", serverIP),
                "--connect-push" , fmt.Sprintf("tcp://%s:7270", serverIP),
                "--storage-url"  , fmt.Sprintf("https://%s", serverHostname),
                "--public-ip"    , curIP,
                "--min-port=7400",
                "--max-port=7700",
                "--heartbeat-interval=10000",
                "--server-ip"    , serverIP,
                "--no-cleanup" )

            outputPipe, _ := cmd.StderrPipe()
            cmd.Dir = "./repos/stf"
            cmd.Stdout = os.Stdout

            backoff.markStart()
            
            err := cmd.Start()
            if err != nil {
                plog.WithFields( log.Fields{
                    "type": "proc_err",
                    "error": err,
                } ).Error("Error starting stf")

                baseProgs.stf = nil
            } else {
                baseProgs.stf = cmd.Process
            }

            scanner := bufio.NewScanner( outputPipe )
            for scanner.Scan() {
                line := scanner.Text()
                if strings.Contains( line, " IOS Heartbeat:" ) {
                } else {
                    lineLog.WithFields( log.Fields{ "line": line } ).Info("")
                }
            }
            
            cmd.Wait()
            backoff.markEnd()

            plog.WithFields( log.Fields{ "type": "proc_end" } ).Warn("Ended: stf_provider")

            if baseProgs.shuttingDown {
                break
            }
            
            backoff.wait()
        }
    }()
}