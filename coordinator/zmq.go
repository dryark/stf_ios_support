package main

import (
  "encoding/json"
  "io"
  "strconv"
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
            }

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

func coro_zmqReqRep() {
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

        for {
            buf := make([]byte, 2000)
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
