package main

import (
  "os/exec"
  "strings"
  "time"
  log "github.com/sirupsen/logrus"
)

func getDeviceName( uuid string ) (string) {
    i := 0
    var nameStr string
    for {
        i++
        if i > 10 { return "" }
        name, _ := exec.Command( "/usr/local/bin/idevicename", "-u", uuid ).Output()
        if name == nil || len(name) == 0 {
            log.WithFields( log.Fields{
                "type": "ilib_getname_fail",
                "uuid": uuid,
                "try": i,
            } ).Debug("idevicename returned nothing")

            time.Sleep( time.Millisecond * 100 )
            continue
        }
        nameStr = string( name )
        break
    }
    nameStr = nameStr[:len(nameStr)-1]
    return nameStr
}

func getAllDeviceInfo( uuid string ) map[string] string {
    rawInfo := getDeviceInfo( uuid, "" )
    lines := strings.Split( rawInfo, "\n" )
    info := make( map[string] string )
    for _, line := range lines {
        char1 := line[0:1]
        if char1 == " " { continue }
        colonPos := strings.Index( line, ":" )
        key := line[0:colonPos]
        val := line[(colonPos+2):]
        info[ key ] = val
    }
    return info
}

func getDeviceInfo( uuid string, keyName string ) (string) {
    i := 0
    var nameStr string
    for {
        i++
        if i > 20 {
            log.WithFields( log.Fields{
                "type": "ilib_getinfo_fail",
                "uuid": uuid,
                "key": keyName,
                "try": i,
            } ).Error("ideviceinfo failed after 20 attempts over 20 seconds")
            return ""
        }

        ops := []string{}
        if uuid != "" {
          ops = append( ops, "-u", uuid )
        }
        if keyName != "" {
          ops = append( ops, "-k", keyName )
        }

        log.WithFields( log.Fields{
            "type": "ilib_getinfo_call",
            "ops": ops,
        } ).Debug("ideviceinfo call")
        name, _ := exec.Command( "/usr/local/bin/ideviceinfo", ops... ).Output()
        if name == nil || len(name) == 0 {
            log.WithFields( log.Fields{
                "type": "ilib_getinfo_fail",
                "uuid": uuid,
                "key":  keyName,
                "try":  i,
            } ).Debug("ideviceinfo returned nothing")

            time.Sleep( time.Millisecond * 1000 )
            continue
        }
        nameStr = string( name )
        break
    }
    nameStr = nameStr[:len(nameStr)-1]
    return nameStr
}