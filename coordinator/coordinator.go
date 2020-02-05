package main

import (
    "flag"
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "sync"
    "syscall"
    "time"
    log "github.com/sirupsen/logrus"
    fsnotify "github.com/fsnotify/fsnotify"
)

type DevEvent struct {
    action int
    uuid   string
    width  int
    height int
}

type RunningDev struct {
    uuid          string
    name          string
    
    // mirrorfeed
    mirror        *os.Process
    mirrorBackoff *Backoff
    
    // ffmpeg
    ff            *os.Process
    ffBackoff     *Backoff
    
    // iproxy to forward localhost:vncPort to phone:5900
    iproxy        *os.Process
    iproxyBackoff *Backoff
    
    wdaWrapper    *Launcher
    device        *os.Process
    deviceBackoff *Backoff
    shuttingDown  bool
    lock          sync.Mutex
    failed        bool
    wdaPort       int
    vidPort       int
    devIosPort    int
    vncPort       int
    pipe          string
    wdaStdoutPipe string
    wdaStderrPipe string
    heartbeatChan chan<- bool
    iosVersion    string
    confDup       *Config
    videoReady    bool
    streamWidth   int
    streamHeight  int
    okFirstFrame  bool
    okVidInterface bool
    okAllUp        bool
}

type BaseProgs struct {
    trigger    *os.Process
    vidEnabler *os.Process
    stf        *os.Process
    shuttingDown bool
    vpnLauncher *Launcher
    vpnLogWatcher *fsnotify.Watcher
    vpnLogWatcherStopChan chan<- bool
    vpnIface   string
    okStage1   bool
    okVpn      bool
}

var gStop bool

func main() {
    gStop = false

    var debug      = flag.Bool( "debug"   , false        , "Use debug log level" )
    var jsonLog    = flag.Bool( "json"    , false        , "Use json log output" )
    var vpnlist    = flag.Bool( "vpnlist" , false        , "List VPNs then exit" )
    var loadVpn    = flag.Bool( "loadVpn" , false        , "Setup / Load OpenVPN plist" )
    var unloadVpn  = flag.Bool( "unloadVpn",false        , "Unload / Remove OpenVPN plist" )
    var load       = flag.Bool( "load"    , false        , "Load Coordinator plist" )
    var unload     = flag.Bool( "unload"  , false        , "Unload Coordinator plist" )
    var addNetPerm = flag.Bool( "addNetPerm", false      , "Add network permission for coordinator app" )
    var getNetPerm = flag.Bool( "getNetPerm", false      , "Show apps with network permission" )
    var delNetPerm = flag.Bool( "delNetPerm", false      , "Delete network permission for coordinator app" )
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
    changeDir := false
    if strings.HasSuffix( dir, "/Contents/MacOS" ) { // running via App
        resDir, _ := filepath.Abs( dir + "/../Resources" )
        configFileA := resDir + "/config.json"
        configFile = &configFileA
        changeDir = true
    }
    
    config := read_config( *configFile )

    if changeDir {
        os.Chdir( config.Install.RootPath )
    }
    
    if *loadVpn {
        openvpn_load( config )
        os.Exit(0)
    }
    if *unloadVpn {
        openvpn_unload( config )
        os.Exit(0)
    }
    
    if *load {
        coordinator_load( config )
        os.Exit(0)
    }
    if *unload {
        coordinator_unload( config )
        os.Exit(0)
    }
    
    if *addNetPerm {
        firewall_ensureperm( "/Applications/STF Coordinator.app" )
        os.Exit(0)
    }
    if *getNetPerm {
        firewall_showperms()
        os.Exit(0)
    }
    if *delNetPerm {
        firewall_delperm( "/Applications/STF Coordinator.app" )
        os.Exit(0)
    }
    
    if config.Install.RootPath != "" {
        os.Chdir( config.Install.RootPath )
    }
    
    useVPN := true
    if config.Vpn.TblickName == "none" && config.Vpn.VpnType != "openvpn" {
        useVPN = false
    }

    baseProgs := BaseProgs{}
    
    vpnEventCh := make( chan VpnEvent, 2 )
    if useVPN {
        check_vpn_status( config, &baseProgs, vpnEventCh )
    }

    lineLog := setup_log( config, *debug, *jsonLog )

    pubEventCh := make( chan PubEvent, 2 )

    runningDevs := make( map [string] *RunningDev )
    
    devEventCh := make( chan DevEvent, 2 )
    
    coro_zmqPull( runningDevs, lineLog, pubEventCh, devEventCh )
    coro_zmqReqRep( runningDevs )
    coro_zmqPub( pubEventCh )

    var ifName     string
    var curIP      string
    var vpnMissing bool
    if useVPN {
        if config.Vpn.VpnType == "tunnelblick" {
            baseProgs.okVpn = true
            baseProgs.okStage1 = true
            ifName, curIP, vpnMissing = get_net_info( config )
        } else if config.Vpn.VpnType == "openvpn" {
            baseProgs.okVpn = false
            baseProgs.okStage1 = false
        }
    } else {
        ifName = config.Network.Iface
        curIP = ifAddr( ifName )
    }

    cleanup_procs( config )

    portMap := NewPortMap( config )
    
    log.WithFields( log.Fields{
        "type":     "portmap",
        "vid_ports": portMap.vidPorts,
        "wda_ports": portMap.wdaPorts,
        "vnc_ports": portMap.vncPorts,
        "dev_ios_ports": portMap.devIosPorts,
    } ).Debug("Portmap")
    
    baseProgs.shuttingDown = false

    coro_http_server( config, devEventCh, &baseProgs, runningDevs )
    proc_device_trigger( config, &baseProgs, lineLog )
    if config.Video.Enabled {
        //ensure_proper_pipe( config )
        proc_video_enabler( config, &baseProgs )
    }

    if useVPN {
        if vpnMissing {
            log.WithFields( log.Fields{ "type": "vpn_warn" } ).Warn("VPN not enabled; skipping start of STF")
            baseProgs.stf = nil
        } else if config.Vpn.VpnType == "tunnelblick" {
            // Start provider here because tunnelblick is connected in the check_vpn_status call above
            // "stage1" is done
            proc_stf_provider( &baseProgs, curIP, config, lineLog )
        }
    } else {
        proc_stf_provider( &baseProgs, curIP, config, lineLog )
    }

    coro_sigterm( runningDevs, &baseProgs, config )

    // process devEvents
    event_loop( config, curIP, devEventCh, vpnEventCh, ifName, pubEventCh, runningDevs, portMap, lineLog, &baseProgs )
}

func coordinator_NewLauncher( config *Config ) (*Launcher) {
    arguments := []string {
        "/Applications/STF Coordinator.app/Contents/MacOS/coordinator",
    }
    
    label := fmt.Sprintf("com.tmobile.coordinator.app")
    wd := "/Applications/STF Coordinator.app/Contents/MacOS"
    keepalive := true
    asRoot := false
    cLauncher := NewLauncher( label, arguments, keepalive, wd, asRoot )
    cLauncher.stdout = config.Log.MainApp
    //cLauncher.stderr = config.WDAWrapperStderr
    
    return cLauncher
}

func coordinator_load( config *Config ) {
    // Check that the App is installed
    // TODO
    
    // Check that the App has video permissions
    bytes, _ := exec.Command( "/usr/bin/perl", "./util/tcc.pl", "hascamera" ).Output()
    if strings.Index( string( bytes ), "no" ) != -1 {
        log.Fatal( "App does not has video permissions; run `sudo utils/tcc.pl addcamera`" )
    }
    
    // Check that the App has network permissions
    hasNetPerm := firewall_hasperm( "/Applications/STF Coordinator.app/Contents/MacOS/coordinator" )
    if !hasNetPerm {
        log.Fatal( "App does not has firewall permissions; run `sudo bin/coordinator -addNetPerm`" )
    }
    
    cLauncher := coordinator_NewLauncher( config )
    cLauncher.load()
}

func coordinator_unload( config *Config ) {
    cLauncher := coordinator_NewLauncher( config )
    cLauncher.unload()
}

func NewRunningDev( 
        gConfig *Config,
        runningDevs map[string] *RunningDev,
        portMap *PortMap,
        uuid string ) ( *RunningDev ) {
        
    wdaPort, vidPort, devIosPort, vncPort, config := assign_ports( gConfig, portMap )
  
    devd := RunningDev{
        uuid: uuid,
        shuttingDown:  false,
        failed:        false,
        mirror:        nil,
        ff:            nil,
        device:        nil,
        deviceBackoff: nil,
        //proxy:         nil,
        wdaPort:       wdaPort,
        vidPort:       vidPort,
        devIosPort:    devIosPort,
        vncPort:       vncPort,
        pipe:          fmt.Sprintf("video_pipes/pipe_%d", vidPort ),
        confDup:       config,
        wdaWrapper:    nil,
        videoReady:    false,
        streamWidth:   0,
        streamHeight:  0,
        okFirstFrame:  false,
        okVidInterface: false,
        okAllUp:        false,
    }
    
    devd.name = getDeviceName( uuid )
    
    runningDevs[uuid] = &devd
    
    log.WithFields( log.Fields{
        "type":     "devd_create",
        "dev_name": devd.name,
        "dev_uuid": uuid,
        "vid_port": vidPort,
        "wda_port": wdaPort,
        "vnc_port": vncPort,
        "dev_ios_port": devIosPort,
    } ).Info("Device object created")
    
    return &devd
}

func event_loop(
        gConfig     *Config,
        curIP       string,
        devEventCh  <-chan DevEvent,
        vpnEventCh  <-chan VpnEvent,
        tunName     string,
        pubEventCh  chan<- PubEvent,
        runningDevs map[string] *RunningDev,
        portMap     *PortMap,
        lineLog     *log.Entry,
        baseProgs *BaseProgs ) {
    
    if baseProgs.okStage1 == false {
        for {
            vpnEvent := <- vpnEventCh
            if vpnEvent.action == 0 {
                tunName = vpnEvent.text1
                curIP = ifAddr( tunName )
                baseProgs.okVpn = true
            }
            
            if baseProgs.okVpn == true && baseProgs.okStage1 == false {
                proc_stf_provider( baseProgs, curIP, gConfig, lineLog )
                baseProgs.okStage1 = true
                break
            }
        }
    }

    for {
        select {
        case vpnEvent := <- vpnEventCh:
            if vpnEvent.action == 0 {
                //iface := vpnEvent.text1
                // do nothing; assume tunnel interface is unchanged
            }
            
        case devEvent := <- devEventCh:
            uuid := devEvent.uuid
    
            var devd *RunningDev = nil
            var ok = false
            
            if devd, ok = runningDevs[uuid]; !ok {
                devd = NewRunningDev( gConfig, runningDevs, portMap, uuid )
            }
            
            if devEvent.action == 0 { // device connect
                devName := devd.name
    
                log.WithFields( log.Fields{
                    "type":     "dev_connect",
                    "dev_name": devName,
                    "dev_uuid": uuid,                
                } ).Info("Device connected")
    
                config := devd.confDup
                
                if config.Video.Enabled {
                    ensure_proper_pipe( devd.pipe )
                    proc_mirrorfeed( config, tunName, devd, lineLog )
                    proc_ffmpeg( config, devd, devName, lineLog )
                }
            }
            if devEvent.action == 1 { // device disconnect
                log.WithFields( log.Fields{
                    "type":     "dev_disconnect",
                    "dev_name": devd.name,
                    "dev_uuid": uuid,
                } ).Info("Device disconnected")
    
                // send true to the stop heartbeat channel
                
                closeRunningDev( devd, portMap )
    
                delete( runningDevs, uuid )
                devd = nil
                
                // Notify stf that the device is gone
                pubEvent := PubEvent{}
                pubEvent.action  = devEvent.action
                pubEvent.uuid    = devEvent.uuid
                pubEvent.name    = ""
                pubEvent.wdaPort = 0
                pubEvent.vidPort = 0
                pubEventCh <- pubEvent
            }
            if devEvent.action == 2 { // video interface available
                log.WithFields( log.Fields{
                    "type":     "vid_interface",
                    "dev_name": devd.name,
                    "dev_uuid": uuid,
                } ).Debug("Video - interface available")
                devd.okVidInterface = true
            }
            if devEvent.action == 3 { // first video frame
                devd.okFirstFrame = true
                devd.streamWidth = devEvent.width
                devd.streamHeight = devEvent.height
                log.WithFields( log.Fields{
                    "type": "first_frame",
                    "proc": "mirrorfeed",
                    "width": devEvent.width,
                    "height": devEvent.height,
                    "uuid": uuid,
                } ).Info("Video - first frame")
            }
            
           if devd != nil && devd.okAllUp == false {
                config := devd.confDup
                
                if !config.Video.Enabled || ( devd.okVidInterface == true && devd.okFirstFrame == true ) {
                    devd.okAllUp = true
                    continue_dev_start( config, devd, curIP, lineLog )
                }
            }
        }
    }
}

func continue_dev_start( config *Config, devd *RunningDev, curIP string, lineLog *log.Entry ) {
    uuid := devd.uuid
    
    if config.Video.Enabled && config.Video.UseVnc {
        proc_vnc_proxy( devd.confDup, devd, lineLog )
    }
    
    time.Sleep( time.Second * 2 )
    
    iosVersion := getDeviceInfo( uuid, "ProductVersion" )

    log.WithFields( log.Fields{
        "type":     "ios_version",
        "dev_name": devd.name,
        "dev_uuid": uuid,
        "ios_version": iosVersion,
    } ).Debug("IOS Version")
    
    start_proc_wdaproxy( devd.confDup, devd, uuid, iosVersion )
    
    time.Sleep( time.Second * 3 )

    proc_device_ios_unit( devd.confDup, devd, uuid, curIP, lineLog )
}

func ensure_proper_pipe( file string ) {
    info, err := os.Stat( file )
    if os.IsNotExist( err ) {
        log.WithFields( log.Fields{
            "type": "pipe_create",
            "pipe_file": file,
        } ).Info("Pipe did not exist; creating as fifo")
        // create the pipe
        syscall.Mkfifo( file, 0600 )
        return
    }
    mode := info.Mode()
    if ( mode & os.ModeNamedPipe ) == 0 {
        log.WithFields( log.Fields{
            "type": "pipe_fix",
            "pipe_file": file,
        } ).Info("Pipe was incorrect type; deleting and recreating as fifo")
        // delete the file then create it properly as a pipe
        os.Remove( file )
        syscall.Mkfifo( file, 0600 )
    }
}