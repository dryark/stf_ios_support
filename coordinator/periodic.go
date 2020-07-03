package main

import (
    "time"  
)

func do_restart( config *Config, devd *RunningDev ) {
    if config.Stf.AdminToken != "" {
        stf_reserve( config, devd.uuid )
    }
    restart_wdaproxy( devd )
    wait_wdaup( devd )
    if config.Stf.AdminToken != "" {
        stf_release( config, devd.uuid )
    }
}

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
                    if devd.owner == "" {
                        do_restart( config, devd )
                    } else {
                        restart_closure := func() { do_restart( config, devd ) }
                        stf_on_release( restart_closure )
                    }
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