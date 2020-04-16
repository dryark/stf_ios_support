package main

import (
  "flag"
  "fmt"
  "encoding/json"
  "os/exec"
  "strconv"
  "strings"
  "time"
  zmq "github.com/zeromq/goczmq"
  log "github.com/sirupsen/logrus"
  gocmd "github.com/go-cmd/cmd"
)

var exit bool
var reqSock *zmq.Sock
var reqOb *zmq.ReadWriter

func main() {
    exit = false

    var wdaPort    = flag.Int( "port", 8100, "WDA Port" )
    var uuid       = flag.String( "uuid", "", "IOS Device UUID" )
    var iosVersion = flag.String( "iosVersion", "", "IOS Version" )
    var wdaRoot    = flag.String( "wdaRoot", "", "WDA Folder Path"    )
    flag.Parse()
    
    setup_zmq()
    proc_wdaproxy( *wdaPort, *uuid, *iosVersion, *wdaRoot )
}


func proc_wdaproxy(
        wdaPort int,
        uuid string,
        iosVersion string,
        wdaRoot string ) {

    log.WithFields( log.Fields{
        "type": "proc_start",
        "proc": "wda_wrapper",
        "wdaPort": wdaPort,
        "uuid": uuid,
        "iosVersion": iosVersion,
        "wdaRoot": wdaRoot,
    } ).Info("wda wrapper started")

    backoff := Backoff{}
    
    for {
        ops := []string{
          "-p", strconv.Itoa( wdaPort ),
          "-q", strconv.Itoa( wdaPort ),
          "-d",
          "-W", ".",
          "-u", uuid,
          fmt.Sprintf("--iosversion=%s", iosVersion),
        }

        log.WithFields( log.Fields{
            "type": "proc_cmdline",
            "cmd": "../wdaproxy",
            "args": ops,
        } ).Info("")
        
        cmd := exec.Command( "../wdaproxy", ops... )

        cmd.Dir = wdaRoot

        stdoutChan := make(chan string, 100)
        stderrChan := make(chan string, 100)
        
        go func() {
           for line := range stdoutChan {

                if strings.Contains( line, "is implemented in both" ) {
                } else if strings.Contains( line, "Couldn't write value" ) {
                } else if strings.Contains( line, "GET /status " ) {
                } else if strings.Contains( line, "] Error" ) {
                    msgCoord( map[string]string{
                      "type": "wda_error",
                      "line": line,
                      "uuid": uuid,
                    } )
                } else {
                    log.WithFields( log.Fields{
                        "type": "proc_stdout",
                        "line": line,
                    } ).Info("")
                    msgCoord( map[string]string{
                      "type": "wda_stdout",
                      "line": line,
                      "uuid": uuid,
                    } )
                    // send line to linelog ( through zmq )
                }
            }
            
            time.Sleep( time.Millisecond * 20 )
        } ()
  
        go func() {
            for line := range stderrChan {
                if strings.Contains( line, "[WDA] successfully started" ) {
                    // send message that WDA has started to coordinator
                    msgCoord( map[string]string{
                      "type": "wda_started",
                      "uuid": uuid,
                    } )
                }
                
                // send line to coordinator
                log.WithFields( log.Fields{
                    "type": "proc_stderr",
                    "line": line,
                } ).Error("")
                msgCoord( map[string]string{
                  "type": "wda_stderr",
                  "line": line,
                  "uuid": uuid,
                } )
                
                time.Sleep( time.Millisecond * 20 )
            }
        } ()
        
        stdStream := gocmd.NewOutputStream(stdoutChan)
        cmd.Stdout = stdStream
        
        errStream := gocmd.NewOutputStream(stderrChan)
        cmd.Stderr = errStream
                
        backoff.markStart()
        err := cmd.Start()
        if err != nil {
            log.WithFields( log.Fields{
                "type": "proc_err",
                "error": err,
            } ).Error("Error starting wdaproxy")
            backoff.markEnd()
            backoff.wait()
            continue
        }
        
        // send message that it has started
        msgCoord( map[string]string{
          "type": "wdaproxy_started",
          "uuid": uuid,
        } )
        
        cmd.Wait()
        
        backoff.markEnd()
        
        // send message that it has ended
        log.WithFields( log.Fields{
            "type": "proc_end",
        } ).Info("Wdaproxy ended")
        msgCoord( map[string]string{
          "type": "wdaproxy_ended",
          "uuid": uuid,
        } )
        
        if exit { break }
        
        backoff.wait()
    }
        
    close_zmq()
}

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

func setup_zmq() {
    reqSock = zmq.NewSock(zmq.Push)
    
    spec := "tcp://127.0.0.1:7300"
    reqSock.Connect( spec )

    var err error
    reqOb, err = zmq.NewReadWriter(reqSock)
    if err != nil {
        log.WithFields( log.Fields{
            "type": "zmq_connect_err",
            "err": err,
        } ).Error("ZMQ Send Error")
    }
    
    reqOb.SetTimeout(1000)
    
    zmqRequest( []byte("dummy") )
}

func close_zmq() {
    reqSock.Destroy()
    reqOb.Destroy()
}

func msgCoord( content map[string]string ) {
    data, _ := json.Marshal( content )
    zmqRequest( data )
}

func zmqRequest( jsonOut []byte ) {
    err := reqSock.SendMessage( [][]byte{ jsonOut } )
    if err != nil {
        log.WithFields( log.Fields{
            "type": "zmq_send_err",
            "err": err,
        } ).Error("ZMQ Send Error")
    }
}