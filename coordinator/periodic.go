package main

import (
    "time"  
)

func periodic_start( config *Config, devd *RunningDev ) {
    endChan := devd.periodic
    wdaRestartMinutes := config.Timing.WdaRestart
    go func() {
        minute := 0
        stop := false
        for {
            time.Sleep( time.Minute * 1 )
            minute++
            if wdaRestartMinutes != 0 {
                if ( minute % wdaRestartMinutes ) == 0 { // every 4 hours by default
                    restart_wdaproxy( devd )
                }
            }
            select {
                case <- endChan:
                    stop = true
                    break
                default:
            }
            if stop { break }
        }
    } ()
}

func periodic_stop( devd *RunningDev ) {
    endChan := devd.periodic
    endChan <- true
}