package main

import (
  "bufio"
  "fmt"
  "os"
  "os/exec"
  "strconv"
  "strings"
  log "github.com/sirupsen/logrus"
)

func proc_device_ios_unit( config *Config, devd *RunningDev, uuid string, curIP string, lineLog *log.Entry ) {
    plog := log.WithFields( log.Fields{
      "proc": "stf_device_ios",
      "uuid": uuid,
    } )
    lineLog = lineLog.WithFields( log.Fields{
        "proc": "stf_device_ios",
        "uuid": devd.uuid,
    } )
    
    backoff := Backoff{}
    devd.deviceBackoff = &backoff

    go func() {
        for {
            plog.WithFields( log.Fields{
              "type": "proc_start",
              "server_ip": config.Stf.Ip,
              "client_ip": curIP,
              "server_host": config.Stf.HostName,
              "video_port": devd.vidPort,
              "node_port": devd.devIosPort,
              "device_name": devd.name,
              "vnc_scale": config.Video.VncScale,
              "stream_width": devd.streamWidth,
              "stream_height": devd.streamHeight,
            } ).Info("Process start - stf_device_ios")

            vncPort := 0
            if config.Video.UseVnc && config.Video.Enabled {
                vncPort = devd.vncPort
            }
            
            cmd := exec.Command( "/usr/local/opt/node@12/bin/node",
                fmt.Sprintf("--inspect=0.0.0.0:%d", devd.devIosPort),
                "runmod.js"              , "device-ios",
                "--serial"               , uuid,
                "--name"                 , devd.name,
                "--connect-push"         , fmt.Sprintf("tcp://%s:7270", config.Stf.Ip),
                "--connect-sub"          , fmt.Sprintf("tcp://%s:7250", config.Stf.Ip),
                "--public-ip"            , curIP,
                "--wda-port"             , strconv.Itoa( devd.wdaPort ),
                "--storage-url"          , fmt.Sprintf("https://%s", config.Stf.HostName),
                "--screen-ws-url-pattern", fmt.Sprintf("wss://%s/frames/%s/%d/x", config.Stf.HostName, curIP, devd.vidPort),
                "--vnc-password"         , config.Video.VncPassword,
                "--vnc-port"             , strconv.Itoa( vncPort ),
                "--vnc-scale"            , strconv.Itoa( config.Video.VncScale ),
                "--stream-width"         , strconv.Itoa( devd.streamWidth ),
                "--stream-height"        , strconv.Itoa( devd.streamHeight ),
            )
            cmd.Dir = "./repos/stf-ios-provider"
            outputPipe, _ := cmd.StderrPipe()
            cmd.Stdout = os.Stdout

            backoff.markStart()
            
            err := cmd.Start()
            if err != nil {
                plog.WithFields( log.Fields{
                    "type": "proc_err",
                    "error": err,
                } ).Error("Error starting device_ios_unit")

                devd.device = nil
            } else {
                devd.device = cmd.Process
            }

            scanner := bufio.NewScanner( outputPipe )
            for scanner.Scan() {
                line := scanner.Text()
                if strings.Contains( line, "Now owned by" ) {
                    pos := strings.Index( line, "Now owned by" )
                    pos += len( "Now owned by" ) + 2
                    ownedStr := line[ pos: ]
                    endpos := strings.Index( ownedStr, "\"" )
                    owner := ownedStr[ :endpos ]
                    plog.WithFields( log.Fields{
                        "type": "wda_owner_start",
                        "owner": owner,
                    } ).Info("Device Owner Start")
                }
                if strings.Contains( line, "No longer owned by" ) {
                    pos := strings.Index( line, "No longer owned by" )
                    pos += len( "No longer owned by" ) + 2
                    ownedStr := line[ pos: ]
                    endpos := strings.Index( ownedStr, "\"" )
                    owner := ownedStr[ :endpos ]
                    plog.WithFields( log.Fields{
                        "type": "wda_owner_stop",
                        "owner": owner,
                    } ).Info("Device Owner Stop")
                }
                if strings.Contains( line, "responding with identity" ) {
                    plog.WithFields( log.Fields{
                        "type": "device_ios_ident",
                    } ).Debug("Device IOS Unit Registered Identity")
                }
                if strings.Contains( line, "Sent ready message" ) {
                    plog.WithFields( log.Fields{
                        "type": "device_ios_ready",
                    } ).Debug("Device IOS Unit Ready")
                }

                lineLog.WithFields( log.Fields{ "line": line } ).Info("")
            }
            
            // Just because output finished doesn't mean the process finished.
            cmd.Wait()

            devd.device = nil
            
            seconds := backoff.markEnd()
              
            plog.WithFields( log.Fields{
                "type": "proc_end",
                "elapsed_sec": seconds,
            } ).Warn("Process end - stf_device_ios")
            
            devd.lock.Lock()
            exit := devd.shuttingDown
            devd.lock.Unlock()
            if exit { break }
            
            backoff.wait()
        }
    }()
}