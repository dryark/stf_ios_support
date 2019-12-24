package main

import "time"

type Backoff struct {
    fails          int
    start          time.Time
    elapsedSeconds float64
}

func ( self *Backoff ) markStart() {
    self.start = time.Now()
}

func ( self *Backoff ) markEnd() ( float64 ) {
    elapsed := time.Since( self.start )
    seconds := elapsed.Seconds()
    self.elapsedSeconds = seconds
    return seconds
}

func ( self *Backoff ) wait() {
    sleeps := []int{ 0, 0, 2, 5, 10 }
    numSleeps := len( sleeps )
    if self.elapsedSeconds < 20 {
        self.fails = self.fails + 1
        index := self.fails
        if index >= numSleeps {
            index = numSleeps - 1
        }
        sleepLen := sleeps[ index ]
        if sleepLen != 0 {
            time.Sleep( time.Second * time.Duration( sleepLen ) )
        }
    } else {
        self.fails = 0
    }
}