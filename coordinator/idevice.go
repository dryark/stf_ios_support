package main

import (
  "os/exec"
  "strings"
  "time"
  log "github.com/sirupsen/logrus"
  uj "github.com/nanoscopic/ujsonin/mod"
)

func getDeviceName( config *Config, uuid string ) (string) {
    i := 0
    var nameStr string
    for {
        i++
        if i > 10 { return "" }
        
        var name []byte
        if config.IosCLI == "ios-deploy" {
            name, _ = exec.Command( config.BinPaths.IosDeploy, "-i", uuid, "-g", "DeviceName" ).Output()
        } else {
            name, _ = exec.Command( "/usr/local/bin/idevicename", "-u", uuid ).Output()
        }
        if name == nil || len(name) == 0 {
            log.WithFields( log.Fields{
                "type": "ilib_getname_fail",
                "uuid": uuid,
                "try":  i,
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

func getAllDeviceInfo( config *Config, uuid string ) map[string] string {
    info := make( map[string] string )
    
    if config.IosCLI == "ios-deploy" {
        mainKeys := "DeviceName,EthernetAddress,ModelNumber,HardwareModel,PhoneNumber,ProductType,ProductVersion,UniqueDeviceID,InternationalCircuitCardIdentity,InternationalMobileEquipmentIdentity,InternationalMobileSubscriberIdentity"
        keyArr := strings.Split( mainKeys, "," )
        output, _ := exec.Command( config.BinPaths.IosDeploy, "-j", "-i", uuid, "-g", mainKeys ).Output()
        root, _ := uj.Parse( output )
        for _, key := range keyArr {
            node := root.Get( key )
            if node != nil {
                info[ key ] = node.String()
            }
        }
        return info
    }
    
    rawInfo := getDeviceInfo( config, uuid, "" )
    lines := strings.Split( rawInfo, "\n" )
    
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

func getDeviceInfo( config *Config, uuid string, keyName string ) (string) {
    i := 0
    var nameStr string
    for {
        i++
        if i > 20 {
            log.WithFields( log.Fields{
                "type": "ilib_getinfo_fail",
                "uuid": uuid,
                "key":  keyName,
                "try":  i,
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
            "ops":  ops,
        } ).Info("ideviceinfo call")
        
        var name []byte
        
        if config.IosCLI == "ios-deploy" {
            if( keyName == "" ) {
                keyName = "DeviceName,EthernetAddress,ModelNumber,HardwareModel,PhoneNumber,ProductType,ProductVersion,UniqueDeviceID,InternationalCircuitCardIdentity,InternationalMobileEquipmentIdentity,InternationalMobileSubscriberIdentity"
            } 
            name, _ = exec.Command( config.BinPaths.IosDeploy, "-i", uuid, "-g", keyName ).Output()
        } else {
            name, _ = exec.Command( "/usr/local/bin/ideviceinfo", ops... ).Output()
        }
        
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

func getFirstDeviceId( config *Config ) ( string ) {
    deviceIds := getDeviceIds( config )
    return deviceIds[0]
}

func getDeviceIds( config *Config ) ( []string ) {
    if config.IosCLI == "ios-deploy" {
        ids := []string{}
        jsonText, _ := exec.Command( config.BinPaths.IosDeploy, "-d", "-j", "-t", "1" ).Output()
        root, _ := uj.Parse( []byte( "[" + string(jsonText) + "]" ) )
        
        root.ForEach( func( evNode *uj.JNode ) {
            ev := evNode.Get("Event").String()
            if ev == "DeviceDetected" {
                dev := evNode.Get("Device")
                ids = append( ids, dev.Get("DeviceIdentifier").String() )
            }
        } )
        return ids
    }
    output, _ := exec.Command( "/usr/local/bin/idevice_id", "-l" ).Output()
    lines := strings.Split( string(output), "\n" )
    return lines
}