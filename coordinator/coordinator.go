package main

import (
    "flag"
    "fmt"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "time"
    log "github.com/sirupsen/logrus"
)

type DevEvent struct {
    action int
    uuid   string
}

type RunningDev struct {
    uuid          string
    name          string
    mirror        *os.Process
    mirrorBackoff *Backoff
    ff            *os.Process
    ffBackoff     *Backoff
    proxy         *os.Process
    proxyBackoff  *Backoff
    device        *os.Process
    deviceBackoff *Backoff
    shuttingDown  bool
    lock          sync.Mutex
    failed        bool
    wdaPort       int
    vidPort       int
    devIosPort    int
    pipe          string
}

type BaseProgs struct {
    trigger    *os.Process
    vidEnabler *os.Process
    stf        *os.Process
    shuttingDown bool
}

var gStop bool

func main() {
    gStop = false

    var debug      = flag.Bool( "debug"   , false        , "Use debug log level" )
    var jsonLog    = flag.Bool( "json"    , false        , "Use json log output" )
    var vpnlist    = flag.Bool( "vpnlist" , false        , "List VPNs then exit" )
    var configFile = flag.String( "config", "config.json", "Config file path"    )
    flag.Parse()

    if *vpnlist {
        vpns := vpns_getall()

        for vpnName, vpn := range vpns {
            fmt.Printf("Name: %s - Autoconnect: %s - %s\n", vpnName, vpn.autoConnect, vpn.state)
        }
        os.Exit(0)
    }

    dir, _ := filepath.Abs( filepath.Dir( os.Args[0] ) )
    if strings.HasSuffix( dir, "/Contents/MacOS" ) { // running via App
        resDir, _ := filepath.Abs( dir + "/../Resources" )
        configFileA := resDir + "/config.json"
        configFile = &configFileA
    }
    
    config := read_config( *configFile )

    if config.RootPath != "" {
        os.Chdir( config.RootPath )
    }
    
    useVPN := true
    if config.VpnName == "none" {
        useVPN = false
    }

    if useVPN {
        check_vpn_status( config )
    }

    lineLog := setup_log( config, *debug, *jsonLog )

    pubEventCh := make( chan PubEvent, 2 )

    coro_zmqReqRep()
    coro_zmqPub( pubEventCh )

    var ifName     string
    var curIP      string
    var vpnMissing bool
    if useVPN {
        ifName, curIP, vpnMissing = get_net_info( config )
    } else {
        ifName = config.NetworkInterface
        curIP = ifAddr( ifName )
    }

    cleanup_procs( config )

    devEventCh := make( chan DevEvent, 2 )
    runningDevs := make( map [string] *RunningDev )
    wdaPorts    := construct_ports( config, config.WDAPorts )
    vidPorts    := construct_ports( config, config.VidPorts )
    devIosPorts := construct_ports( config, config.DevIosPorts ) 
    baseProgs := BaseProgs{}
    baseProgs.shuttingDown = false

    coro_http_server( config, devEventCh, &baseProgs, runningDevs )
    proc_device_trigger( config, &baseProgs )
    if !config.SkipVideo {
        //ensure_proper_pipe( config )
        proc_video_enabler( config, &baseProgs )
    }

    if useVPN {
        if vpnMissing {
            log.WithFields( log.Fields{ "type": "vpn_warn" } ).Warn("VPN not enabled; skipping start of STF")
            baseProgs.stf = nil
        } else {
            // start stf and restart it when needed
            // TODO: if it doesn't restart / crashes again; give up
            proc_stf_provider( &baseProgs, curIP, config, lineLog )
        }
    } else {
        proc_stf_provider( &baseProgs, curIP, config, lineLog )
    }

    coro_sigterm( runningDevs, &baseProgs, config )

    // process devEvents
    event_loop( config, curIP, devEventCh, ifName, pubEventCh, runningDevs, wdaPorts, vidPorts, devIosPorts, lineLog )
}

func event_loop(
        gConfig     *Config,
        curIP       string,
        devEventCh  <-chan DevEvent,
        tunName     string,
        pubEventCh  chan<- PubEvent,
        runningDevs map[string] *RunningDev,
        wdaPorts    map[int] *PortItem,
        vidPorts    map[int] *PortItem,
        devIosPorts map[int] *PortItem,
        lineLog     *log.Entry ) {
    for {
        // receive message
        devEvent := <- devEventCh
        uuid := devEvent.uuid

        if devEvent.action == 0 { // device connect
            wdaPort, vidPort, devIosPort, config := assign_ports( gConfig, wdaPorts, vidPorts, devIosPorts )
            
            devd := RunningDev{
                uuid: uuid,
                shuttingDown:  false,
                failed:        false,
                mirror:        nil,
                ff:            nil,
                device:        nil,
                deviceBackoff: nil,
                proxy:         nil,
                wdaPort:       wdaPort,
                vidPort:       vidPort,
                devIosPort:    devIosPort,
                pipe:          fmt.Sprintf("video_pipes/pipe_%d", vidPort ),
            }
            runningDevs[uuid] = &devd
            
            devd.name = getDeviceName( uuid )
            if devd.name == "" {
                devd.failed = true
                // TODO log an error
                continue
            }
            devName := devd.name

            log.WithFields( log.Fields{
                "type":     "dev_connect",
                "dev_name": devName,
                "dev_uuid": uuid,
                "vid_port": vidPort,
                "wda_port": wdaPort,
            } ).Info("Device connected")

            if !config.SkipVideo {
                ensure_proper_pipe( &devd )
                proc_mirrorfeed( config, tunName, &devd, lineLog )
                proc_ffmpeg( config, &devd, devName, lineLog )

                // Sleep to ensure that video enabling process is finished before we try to start wdaproxy
                // This is needed because the USB device drops out and reappears during video enabling
                time.Sleep( time.Second * 9 )
            }

            iosVersion := getDeviceInfo( uuid, "ProductVersion" )
            proc_wdaproxy( config, &devd, &devEvent, uuid, devName, pubEventCh, lineLog, iosVersion )

            time.Sleep( time.Second * 3 )

            proc_device_ios_unit( config, &devd, uuid, curIP, lineLog )
        }
        if devEvent.action == 1 { // device disconnect
            devd := runningDevs[uuid]

            log.WithFields( log.Fields{
                "type":     "dev_disconnect",
                "dev_name": devd.name,
                "dev_uuid": uuid,
            } ).Info("Device disconnected")

            closeRunningDev( devd, wdaPorts, vidPorts, devIosPorts )

            // Notify stf that the device is gone
            pubEvent := PubEvent{}
            pubEvent.action  = devEvent.action
            pubEvent.uuid    = devEvent.uuid
            pubEvent.name    = ""
            pubEvent.wdaPort = 0
            pubEvent.vidPort = 0
            pubEventCh <- pubEvent
        }
    }
}