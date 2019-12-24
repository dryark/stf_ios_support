package main

import (
  "os/exec"
  log "github.com/sirupsen/logrus"
)

func proc_device_trigger( config *Config, baseProgs *BaseProgs ) {
    plog := log.WithFields( log.Fields{ "proc": "device_trigger" } )

    go func() {
        plog.WithFields( log.Fields{ "type": "proc_start" } ).Info("Starting: device_trigger")

        triggerCmd := exec.Command( config.DeviceTrigger )

        err := triggerCmd.Start()
        if err != nil {
            plog.WithFields( log.Fields{
                "type": "proc_err",
                "error": err,
            } ).Error("Error starting device_trigger")

            baseProgs.trigger = nil
        } else {
            baseProgs.trigger = triggerCmd.Process
        }

        triggerCmd.Wait()

        plog.WithFields( log.Fields{ "type": "proc_end" } ).Warn("Ended: device_trigger")
    }()
}