package main

import (
  "time"
  //log "github.com/sirupsen/logrus"
)

func coro_heartbeat( uuid string, pubEventCh chan<- PubEvent ) ( chan<- bool ) {
    count := 1
    stopChannel := make(chan bool)

    // Start heartbeat
    go func() {
        done := false
        for {
            select {
                case _ = <-stopChannel:
                    done = true
                default:
            }
            if done {
                break
            }

            if count >= 10 {
                count = 0

                beatEvent := PubEvent{}
                beatEvent.action  = 2
                beatEvent.uuid    = uuid
                beatEvent.name    = ""
                beatEvent.wdaPort = 0
                beatEvent.vidPort = 0
                pubEventCh <- beatEvent
                
                /*log.WithFields( log.Fields{
                    "type": "heartbeat",
                } ).Info("Heartbeat")*/
            }
            time.Sleep( time.Second * 1 )
            count++;
        }
    }()

    return stopChannel
}