package main

import (
  "bytes"
  "encoding/json"
  "fmt"
  "io"
  "strconv"
  "sync"
  "time"
  log "github.com/sirupsen/logrus"
  zmq "github.com/zeromq/goczmq"
)

type PubEvent struct {
    action  int
    uuid    string
    name    string
    wdaPort int
    vidPort int
}

func coro_zmqPub( pubEventCh <-chan PubEvent ) {
    plog := log.WithFields( log.Fields{ "coro": "pub" } )

    var sentDummy bool = false

    // start the zmq pub mechanism
    go func() {
        pubSock := zmq.NewSock(zmq.Pub)
        defer pubSock.Destroy()

        spec := "tcp://127.0.0.1:7294"
        _, err := pubSock.Bind( spec )
        if err != nil {
            plog.WithFields( log.Fields{
                "type": "err_zmq",
                "zmq_spec": spec,
                "err": err,
            } ).Fatal("ZMQ binding error")
        }

        // Garbage message with delay to avoid late joiner ZeroMQ madness
        if !sentDummy {
            pubSock.SendMessage( [][]byte{ []byte("devEvent"), []byte("dummy") } )
            time.Sleep( time.Millisecond * 300 )
        }

        for {
            // receive message
            pubEvent := <- pubEventCh

            //uuid := devEvent.uuid
            type DevTest struct {
                Type    string
                UUID    string
                Name    string
                VidPort string
                WDAPort string
            }
            test := DevTest{}
            test.UUID    = pubEvent.uuid
            test.Name    = pubEvent.name
            test.VidPort = strconv.Itoa( pubEvent.vidPort )
            test.WDAPort = strconv.Itoa( pubEvent.wdaPort )

            if pubEvent.action == 0 {
                test.Type = "connect"
            } else if pubEvent.action == 2 {
                test.Type = "heartbeat"
            } else if pubEvent.action == 1 {
                test.Type = "disconnect"
            } else if pubEvent.action == 3 {
                test.Type = "present"
            }

            /*log.WithFields( log.Fields{
                "type": "zmq_push",
                "event": test,
            } ).Info("ZMQ Push")*/
            
            // publish a zmq message of the DevEvent
            reqMsg, err := json.Marshal( test )
            if err != nil {
                plog.WithFields( log.Fields{
                    "type": "err_zmq_encode",
                    "err": err,
                } ).Error("ZMQ JSON encode error")
            } else {
                plog.WithFields( log.Fields{
                    "type": "zmq_pub",
                    "msg": reqMsg,
                } ).Debug("Publishing to stf")

                pubSock.SendMessage( [][]byte{ []byte("devEvent"), reqMsg} )
            }
        }
    }()
}

func reqDevInfoJSON( uuid string ) (string) {
    res := ""
    info := getAllDeviceInfo( uuid )

    names := map[string] string {
        "DeviceName":      "Device Name",
        "EthernetAddress": "MAC",
        "ModelNumber":     "Model",
        //"HardwareModel": "Hardware Model",
        "PhoneNumber":     "Phone Number",
        "ProductType":     "Product",
        "ProductVersion":  "IOS Version",
        "UniqueDeviceID":  "Wifi MAC",
        "InternationalCircuitCardIdentity":      "ICCI",
        "InternationalMobileEquipmentIdentity":  "IMEI",
        "InternationalMobileSubscriberIdentity": "IMSI",
        
    }

    for key, descr := range names {
        val := info[key]
        res += fmt.Sprintf( "%s: %s<br>\n", descr, val )
    }
    
    return res
}

func devListJSON( runningDevs map[string] *RunningDev, devMapLock *sync.Mutex ) (string) {
    devOut := ""
    devMapLock.Lock()
    for _, dev := range runningDevs {
        mirror := "<font color='green'>on</font>"
        if dev.process["mirror"] == nil { mirror = "off" }

        //proxy := "<font color='green'>on</font>"
        //if dev.proxy == nil { proxy = "off" }

        device := "<font color='green'>on</font>"
        if dev.process["device"] == nil { device = "off" }

        var str bytes.Buffer
        deviceTpl.Execute( &str, map[string] string {
            "uuid":   "<a href='/devinfo?uuid=" + dev.uuid + "'>" + dev.uuid + "</a>",
            "name":   dev.name,
            "mirror": mirror,
            //"proxy":  proxy,
            "device": device,
        } )
        devOut += str.String()
    }
    devMapLock.Unlock()
    return devOut
}

func coro_zmqPull( runningDevs map[string] *RunningDev, devMapLock *sync.Mutex, lineLog *log.Entry, pubEventCh  chan<- PubEvent, devEventCh chan<- DevEvent ) {
    plog := log.WithFields( log.Fields{ "coro": "zmqpull" } )
    
    wdaLineLog := lineLog.WithFields( log.Fields{
        "proc": "wdaproxy",
    } )
    
    go func() {
        pullSock := zmq.NewSock(zmq.Pull)
        defer pullSock.Destroy()
        
        spec := "tcp://127.0.0.1:7300"
        
        _, err := pullSock.Bind( spec )
        if err != nil {
            plog.WithFields( log.Fields{
                "type": "err_zmq",
                "zmq_spec": spec,
                "err": err,
            } ).Fatal("ZMQ binding error")
        }
        
        pullOb, err := zmq.NewReadWriter(pullSock)
        if err != nil {
            plog.WithFields( log.Fields{
                "type": "err_zmq",
                "zmq_spec": spec,
                "err": err,
            } ).Fatal("error making readwriter")
        }
        defer pullOb.Destroy()
        
        pullOb.SetTimeout(1000)
        
        buf := make([]byte, 20000)
        
        for {
            readBytes, err := pullOb.Read( buf )
            if err == zmq.ErrTimeout {
                if gStop == true {
                    break
                }
                continue
            }
            if err != nil && err != io.EOF {
                plog.WithFields( log.Fields{
                    "type": "err_zmq",
                    "err": err,
                } ).Error("Error reading zmq")
            } else {
                if buf[0] == "{"[0] {
                    part := buf[0:readBytes]
                    // Receiving a JSON message
                    var msg map[string]string
                    json.Unmarshal( part, &msg )
                    uuid := msg["uuid"]
                    msgType := msg["type"]
                    if msgType == "wdaproxy_started" {
                        plog.WithFields( log.Fields{
                            "type": "zmq_wdaproxy_started",
                            "proc": "wdaproxy",
                            "uuid": censor_uuid( uuid ),
                        } ).Info("WDA Wrapper - WDAProxy Started")
                        devMapLock.Lock()
                        devd := runningDevs[ uuid ]
                        if devd != nil {
                            devMapLock.Unlock()
                            
                            // Everything is started; notify stf via zmq published event
                            pubEvent := PubEvent{}
                            pubEvent.action  = 0
                            pubEvent.uuid    = msg["uuid"]
                            
                            devd.lock.Lock()
                            devName := devd.name
                            wdaPort := devd.wdaPort
                            vidPort := devd.vidPort
                            devd.lock.Unlock()
                            
                            pubEvent.name    = devName
                            pubEvent.wdaPort = wdaPort
                            pubEvent.vidPort = vidPort
                            pubEventCh <- pubEvent
                            
                            pubEvent = PubEvent{}
                            pubEvent.action  = 3
                            pubEvent.uuid    = msg["uuid"]
                            pubEvent.name    = devName
                            pubEvent.wdaPort = wdaPort
                            pubEvent.vidPort = vidPort
                            pubEventCh <- pubEvent
                        }
                    } else if msgType == "wda_started" {
                        plog.WithFields( log.Fields{
                            "type": "wda_started",
                            "proc": "wdaproxy",
                            "uuid": censor_uuid( uuid ),
                        } ).Info("WDA Running")
                        
                        devEvent := DevEvent{
                            action: 4,
                            uuid: uuid,
                        }
                        
                        devEventCh <- devEvent
                    } else if msgType == "wda_stdout" {
                        wdaLineLog.WithFields( log.Fields {
                            "line": msg["line"],
                            "uuid": uuid,
                        } ).Info("")
                    } else if msgType == "wda_stderr" {
                        wdaLineLog.WithFields( log.Fields {
                            "line": msg["line"],
                            "uuid": uuid,
                        } ).Info("")
                    } else if msgType == "wda_error" {
                        plog.WithFields( log.Fields{
                            "type": "wda_error",
                            "proc": "wdaproxy",
                            "line": msg["line"],
                            "uuid": censor_uuid( uuid ),
                        } ).Error("WDA Error")
                    } else if msgType == "wdaproxy_ended" {
                        devMapLock.Lock()
                        devd := runningDevs[ uuid ]
                        devMapLock.Unlock()
                        
                        if devd == nil {
                        } else {
                            devd.lock.Lock()
                            devd.heartbeatChan <- true
                            devd.lock.Unlock()
                            
                            plog.WithFields( log.Fields{
                                "type": "wdaproxy_ended",
                                "proc": "wdaproxy",
                                "uuid": censor_uuid( uuid ),
                            } ).Error("WDA Wrapper - WDAProxy")
                        }
                    } else if msgType == "mirrorfeed_dimensions" {
                        width, _ := strconv.Atoi( msg["width"] )
                        height, _ := strconv.Atoi( msg["height"] )
                        devEvent := DevEvent{
                            action: 3,
                            width: width,
                            height: height,
                            uuid: uuid,
                        }
                        
                        devEventCh <- devEvent
                    } else {
                        plog.WithFields( log.Fields{
                            "type": "zmq_type_err",
                            "msgType": msgType,
                            "uuid": uuid,
                            "rawMsg": string( part ),
                        } ).Error("Unknown zmq message type")
                    }
                } else {
                   // error
                }
            }
        }
    }()
}

func coro_zmqReqRep( runningDevs map[string] *RunningDev ) {
    plog := log.WithFields( log.Fields{ "coro": "reqrep" } )
    
    go func() {
        repSock := zmq.NewSock(zmq.Rep)
        defer repSock.Destroy()

        spec := "tcp://127.0.0.1:7293"
        _, err := repSock.Bind( spec )
        if err != nil {
            plog.WithFields( log.Fields{
                "type": "err_zmq",
                "zmq_spec": spec,
                "err": err,
            } ).Fatal("ZMQ binding error")
        }

        repOb, err := zmq.NewReadWriter(repSock)
        if err != nil {
            plog.WithFields( log.Fields{
                "type": "err_zmq",
                "zmq_spec": spec,
                "err": err,
            } ).Fatal("error making readwriter")
        }
        defer repOb.Destroy()

        repOb.SetTimeout(1000)

        buf := make([]byte, 20000)
        
        for {
            _, err := repOb.Read( buf )
            if err == zmq.ErrTimeout {
                if gStop == true {
                    break
                }
                continue
            }
            if err != nil && err != io.EOF {
                plog.WithFields( log.Fields{
                    "type": "err_zmq",
                    "err": err,
                } ).Error("Error reading zmq")
            } else {
                msg := string( buf )

                if msg == "quit" {
                    response := []byte("quitting")
                    repSock.SendMessage([][]byte{response})
                    break
                } else if msg == "devices" {
                    //devInfoJSON := reqDevInfoJSON( "" )
                    // TODO: get device list
                    // TOOO: turn device list into JSON

                    response := []byte("quitting")
                    repSock.SendMessage([][]byte{response})
                } else {
                    plog.WithFields( log.Fields{
                        "type": "err_zmq",
                        "msg": string( buf ),
                    } ).Error("Received unknown message")

                    response := []byte("response")
                    repSock.SendMessage([][]byte{response})
                }
            }
        }
    }()
}
