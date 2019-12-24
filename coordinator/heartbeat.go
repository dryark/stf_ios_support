package main

import "time"

func coro_heartbeat( devEvent *DevEvent, pubEventCh chan<- PubEvent ) ( chan<- bool ) {
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
                beatEvent.uuid    = devEvent.uuid
                beatEvent.name    = ""
                beatEvent.wdaPort = 0
                beatEvent.vidPort = 0
                pubEventCh <- beatEvent
            }
            time.Sleep( time.Second * 1 )
            count++;
        }
    }()

    return stopChannel
}