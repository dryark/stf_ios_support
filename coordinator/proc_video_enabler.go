package main

import (
  "os/exec"
  log "github.com/sirupsen/logrus"
)

func proc_video_enabler( config *Config, baseProgs *BaseProgs ) {
    plog := log.WithFields( log.Fields{ "proc": "video_enabler" } )

    go func() {
        plog.WithFields( log.Fields{ "type": "proc_start" } ).Info("Process start - video_enabler")

        enableCmd := exec.Command(config.VideoEnabler)
        err := enableCmd.Start()
        if err != nil {
            plog.WithFields( log.Fields{
                "type": "proc_err",
                "error": err,
            } ).Error("Error starting video_enabler")

            baseProgs.vidEnabler = nil
        } else {
            baseProgs.vidEnabler = enableCmd.Process
        }
        enableCmd.Wait()

        plog.WithFields( log.Fields{ "type": "proc_end" } ).Warn("Process end - video_enabler")
    }()
}