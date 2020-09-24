package main

import (
	"time"
	"strconv"
)

type Backoff struct {
    fails          int
    start          time.Time
    elapsedSeconds float64
}

func ( self *Backoff ) markStart() {
    self.start = time.Now()
}

func ( self *Backoff ) timeUp() ( float64 ) {
	elapsed := time.Since( self.start )
    seconds := elapsed.Seconds()
    return seconds
}

func ( self *Backoff ) timeUpText() ( string ) {
	seconds := uint16( self.timeUp() )
	minutes := uint16(0)
	hours := uint16(0)
	days := uint16(0)
	if seconds > 60 {
		mod := seconds % 60
		minutes = seconds / 60
		seconds = mod
	}
	if minutes > 60 {
		mod := minutes % 60
		hours = minutes / 60
		minutes = mod
	}
	if hours > 24 {
		mod := hours % 24
		days = hours / 24
		hours = mod
	}
	text := strconv.Itoa(int(seconds)) + " sec"
	if minutes > 0 {
		text = strconv.Itoa(int(minutes)) + " mins " + text
	}
	if hours > 0 {
		text = strconv.Itoa(int(hours)) + " hrs " + text
	}
	if days > 0 {
		text = strconv.Itoa(int(days)) + " days " + text
    }
    return text
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