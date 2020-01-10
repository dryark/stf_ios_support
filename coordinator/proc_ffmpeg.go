package main

import (
  "bufio"
  "os"
  "os/exec"
  "strconv"
  "strings"
  "time"
  log "github.com/sirupsen/logrus"
)

func proc_ffmpeg( config *Config, devd *RunningDev, devName string, lineLog *log.Entry ) {
    plog := log.WithFields( log.Fields{
        "proc": "ffmpeg",
        "uuid": devd.uuid,
    } )
    lineLog = lineLog.WithFields( log.Fields{
        "proc": "ffmpeg",
        "uuid": devd.uuid,
    } )

    if devd.shuttingDown {
        return
    }
    
    backoff := Backoff{}
    devd.ffBackoff = &backoff
    
    go func() {
        for {
            ops := []string {
                "-f", "avfoundation",
                "-i", devName,
                "-pixel_format", "bgr0",
                "-f",            "mjpeg",
                "-bsf:v",        "mjpegadump",
                "-bsf:v",        "mjpeg2jpeg",
                "-r",            strconv.Itoa( config.FrameRate ), // framerate
                "-vsync",        "2",
                "-nostats",
                "pipe:1",
            }

            plog.WithFields( log.Fields{
                "type": "proc_start",
                "ops": ops,
            } ).Info("Process start - ffmpeg")

            cmd := exec.Command( "bin/ffmpeg", ops... )
            
            outputPipe, _ := cmd.StderrPipe()
            //cmd.Stdout = os.Stdout
            videoPipe, vidErr := os.OpenFile( devd.pipe, os.O_RDWR|os.O_CREATE|os.O_APPEND, os.ModeNamedPipe )
            if vidErr != nil {
                plog.WithFields( log.Fields{
                    "type": "pipe_err",
                    "error": vidErr,
                } ).Error("Error opening pipe")
                
                time.Sleep( time.Second * 5 )
                
                continue
            }
            cmd.Stdout = videoPipe
            
            backoff.markStart()
            
            err := cmd.Start()
            if err != nil {
                plog.WithFields( log.Fields{
                    "type": "proc_err",
                    "error": err,
                } ).Error("Error starting ffmpeg")

                devd.ff = nil
            } else {
                devd.ff = cmd.Process
            }

            scanner := bufio.NewScanner( outputPipe )
            for scanner.Scan() {
                line := scanner.Text()

                if strings.Contains( line, "avfoundation @" ) {
                } else if strings.Contains( line, "swscaler @" ) {
                } else if strings.HasPrefix( line, "  lib" ) {
                } else {
                    lineLog.WithFields( log.Fields{ "line": line } ).Info("")
                }
            }
            
            cmd.Wait()
            backoff.markEnd()

            plog.WithFields( log.Fields{ "type": "proc_end" } ).Warn("Process end - ffmpeg")

            outputPipe.Close()
            devd.ff = nil

            devd.lock.Lock()
            exit := devd.shuttingDown
            devd.lock.Unlock()
            if exit { break }
            
            backoff.wait()
        }
    }()
}