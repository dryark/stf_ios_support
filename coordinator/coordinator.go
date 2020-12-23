package main

import (
    "bytes"
    "flag"
    "fmt"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "strconv"
    "strings"
    "sync"
    "time"
    log "github.com/sirupsen/logrus"
    fsnotify "github.com/fsnotify/fsnotify"
    uj "github.com/nanoscopic/ujsonin/mod"
)

type DevEvent struct {
    action     int
    uuid       string
    width      int
    height     int
    clickScale int
}

func (self *RunningDev) dup() ( *RunningDev ) {
    if self == nil { return nil }
    self.lock.Lock()
    dup := *self
    self.lock.Unlock()
    return &dup
}

func (self *RunningDev) getShuttingDown( base *BaseProgs ) (bool){
    if self == nil {
        base.lock.Lock()
        sd := base.shuttingDown
        base.lock.Unlock()
        return sd
    }
    self.lock.Lock()
    sd := self.shuttingDown
    self.lock.Unlock()
    return sd
}

type RunningDev struct {
    uuid           string
    name           string
    wdaWrapper     *Launcher
    shuttingDown   bool
    lock           *sync.Mutex
    failed         bool
    wdaPort        int
    vidPort        int
    devIosPort     int
    vncPort        int
    usbmuxdPort    int
    wdaStdoutPipe  string
    wdaStderrPipe  string
    heartbeatChan  chan<- bool
    iosVersion     string
    confDup        *Config
    videoReady     bool
    streamWidth    int
    streamHeight   int
    streamPort     int
    clickWidth     int
    clickHeight    int
    clickScale     int
    okFirstFrame   bool
    okVidInterface bool
    wdaStarted     bool
    process        map[string] *GenericProc
    devUnitStarted bool
    periodic       chan bool
    owner          string
    wda            bool
}

type BaseProgs struct {
    process               map[string] *GenericProc
    shuttingDown          bool
    vpnLauncher           *Launcher
    vpnLogWatcher         *fsnotify.Watcher
    vpnLogWatcherStopChan chan<- bool
    vpnIface              string
    okStage1              bool
    okVpn                 bool
    lock                  *sync.Mutex
}

var gStop       bool

var GitCommit   string
var GitDate     string
var GitRemote   string
var EasyVersion string

func main() {
    gStop = false

    var debug      = flag.Bool( "debug"     , false        , "Use debug log level" )
    var jsonLog    = flag.Bool( "json"      , false        , "Use json log output" )
    var vpnlist    = flag.Bool( "vpnlist"   , false        , "List VPNs then exit" )
    
    var loadVpn    = flag.Bool( "loadVpn"   , false        , "Setup / Load OpenVPN plist" )
    var vpnFile    = flag.String("vpnFile"  , ""           , "OpenVPN file to user" )
    var vpnLabel   = flag.String("vpnLabel" , ""           , "Plist label to use for VPN" )
    
    var unloadVpn  = flag.Bool( "unloadVpn" , false        , "Unload / Remove OpenVPN plist" )
    var load       = flag.Bool( "load"      , false        , "Load Coordinator plist" )
    var unload     = flag.Bool( "unload"    , false        , "Unload Coordinator plist" )
    var addNetPerm = flag.Bool( "addNetPerm", false        , "Add network permission for coordinator app" )
    var getNetPerm = flag.Bool( "getNetPerm", false        , "Show apps with network permission" )
    var delNetPerm = flag.Bool( "delNetPerm", false        , "Delete network permission for coordinator app" )
    var configFile = flag.String( "config"  , "config.json", "Config file path" )
    var testVideo  = flag.Bool( "testVideo" , false        , "Test Video Streaming" )
    var resetVideo = flag.Bool( "resetVideo", false        , "Reset Media Services on device" )
    var doUnlock   = flag.Bool( "unlock"    , false        , "Unlock the IOS device" )
    var doVersion  = flag.Bool( "version"   , false        , "Show coordinator version info" )
    var killProcs  = flag.Bool( "killProcs" , false        , "Terminate leftover processes" )
    
    var reserve    = flag.Bool( "reserve"   , false        , "Reserve device in STF" )
    var release    = flag.Bool( "release"   , false        , "Release device in STF" )
    var ruuid      = flag.String( "udid"    , ""           , "UDID of device to reserve/release" )
    
    flag.Parse()

    if *doVersion {
        fmt.Printf("Commit:%s\nDate:%s\nRemote:%s\nVersion:%s\n", GitCommit, GitDate, GitRemote, EasyVersion )
        os.Exit(0)
    }
    
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
    
    if *debug { fmt.Printf("Loading config\n") }
    config := read_config( *configFile )
    if *debug { fmt.Printf("Config loaded\n") }
    
    if *killProcs {
    	cleanup_procs( config )
    	os.Exit(0)
    }
    
    if changeDir || config.Install.SetWorkingDir {
        os.Chdir( config.Install.RootPath )
    }
    
    if *reserve {
        res := stf_reserve( config, *ruuid )
        if res {
            fmt.Printf("success\n")
        } else {
            fmt.Printf("failure\n")
        }
        os.Exit(0)
    }
    if *release {
        res := stf_release( config, *ruuid )
        if res {
            fmt.Printf("success\n")
        } else {
            fmt.Printf("failure\n")
        }
        os.Exit(0)
    }
    
    if *loadVpn {
        openvpn_load( config, *vpnFile, *vpnLabel )
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
        if *debug { fmt.Printf("Changing directory to %s\n", config.Install.RootPath ) }
        os.Chdir( config.Install.RootPath )
    }
    
    useVPN := true
    if( config.Vpn.TblickName == "none" && config.Vpn.VpnType != "openvpn" ) || config.Vpn.VpnType == "none" {
        useVPN = false
    }
    
    videoMethod := config.Video.Method

    baseProgs := BaseProgs{
        process: make( map[string] *GenericProc ),
        lock: &sync.Mutex{},
    }
    
    lineLog, lineTracker := setup_log( config, *debug, *jsonLog )
    
    vpnEventCh := make( chan VpnEvent, 2 )
    if useVPN {
        log.Debug("Checking VPN status")
        check_vpn_status( config, &baseProgs, vpnEventCh )
    }

    pubEventCh := make( chan PubEvent )

    runningDevs := make( map [string] *RunningDev )
    var devMapLock sync.Mutex
    
    portMap := NewPortMap( config )
    
    devEventCh := make( chan DevEvent )
    
    if *doUnlock {
        devId := getFirstDeviceId( config )
        fmt.Printf("First device ID: %s\n", devId )
        
        devd := NewRunningDev( config, runningDevs, &devMapLock, portMap, devId )
        
        o := ProcOptions {
            config:    config,
            baseProgs: &baseProgs,
            lineLog:   lineLog,
            devd:      devd,
            curIP:     "127.0.0.1",
        }
        
        wda := NewTempWDA( o )
        
        time.Sleep( time.Second * 5 )
        
        isLocked := wda.is_locked()
        if isLocked {
            fmt.Println("Locked; unlocking...")
            wda.unlock()
            fmt.Println("Unlocked")
        } else {
            fmt.Println("Already unlocked")
        }
        
        os.Exit(0)
    }
    
    if *testVideo {
        devId := getFirstDeviceId( config )
        fmt.Printf("First device ID: %s\n", devId )
        
        devd := NewRunningDev( config, runningDevs, &devMapLock, portMap, devId )
        
        o := ProcOptions {
            config:    config,
            baseProgs: &baseProgs,
            lineLog:   lineLog,
            devd:      devd,
            curIP:     "127.0.0.1",
        }
    
        fmt.Printf( "First device name: %s\n", devd.name )
        if videoMethod == "ivp" {
            ivp_enable( o )
        }
                
        proc_ios_video_stream( o, "none", "127.0.0.1" )
        
        if videoMethod == "avfoundation" {
        	proc_video_enabler( o )
            proc_ivf( o )
        } else if videoMethod == "ivp" {
            proc_h264_to_jpeg( o )
            proc_ios_video_pull( o )
        }
        
        coro_sigterm( runningDevs, &baseProgs, config )
        coro_mini_http_server( config, devEventCh, devd )
        /*for {
            shuttingDown := devd.getShuttingDown( o.baseProgs )
            if shuttingDown { break }
            time.Sleep( time.Second * 2 )
        }*/
        mini_event_loop( devEventCh, devd )
        
        os.Exit(0)
    }
    
    if *resetVideo {
    	devId := getFirstDeviceId( config )
        fmt.Printf("First device ID: %s\n", devId )
        
        devd := NewRunningDev( config, runningDevs, &devMapLock, portMap, devId )
        
        o := ProcOptions {
            config:    config,
            baseProgs: &baseProgs,
            lineLog:   lineLog,
            devd:      devd,
            curIP:     "127.0.0.1",
        }
        
    	aio_reset_media_services( o )
    	
    	os.Exit(0)
    }
    
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
        if ifName == "auto" {
        	ifName = getDefaultIf()
        	if ifName == "" {
        		fmt.Println("auto network interface specified but could not determine default interface")
        		os.Exit(1)
        	}
        	log.WithFields( log.Fields{
				"type": "default_iface",
				"interface_name": ifName,
			} ).Info( "auto network interface set; using ", ifName )
        }
        
        var okay bool
        curIP, okay = ifAddr( ifName )
        if !okay {
        	os.Exit(0)
        }
        baseProgs.okStage1 = true
    }
    
    cleanup_procs( config )

    log.Debug("Starting ZMQ Pull")
    coro_zmqPull( runningDevs, &devMapLock, lineLog, pubEventCh, devEventCh )
    log.Debug("Starting ZMQ ReqRep")
    coro_zmqReqRep( runningDevs )
    log.Debug("Starting ZMQ Pub")
    coro_zmqPub( pubEventCh )


    log.WithFields( log.Fields{
        "type":          "portmap",
        "vid_ports":     portMap.vidPorts,
        "wda_ports":     portMap.wdaPorts,
        "vnc_ports":     portMap.vncPorts,
        "usbmuxd_ports": portMap.usbmuxdPorts,
        "dev_ios_ports": portMap.devIosPorts,
    } ).Debug("Portmap")
    
    baseProgs.shuttingDown = false

    procOptions := ProcOptions {
        config:    config,
        baseProgs: &baseProgs,
        lineLog:   lineLog,
        curIP:     curIP,
    }   
    
    coro_http_server( config, devEventCh, &baseProgs, runningDevs, lineTracker )
    proc_device_trigger( procOptions )
    
    if useVPN {
        if vpnMissing {
            log.WithFields( log.Fields{ "type": "vpn_warn" } ).Warn("VPN not enabled; skipping start of STF")
        } else if config.Vpn.VpnType == "tunnelblick" {
            // Start provider here because tunnelblick is connected in the check_vpn_status call above
            // "stage1" is done
            proc_stf_provider( procOptions, curIP )
        }
    } else {
        proc_stf_provider( procOptions, curIP )
    }

    coro_sigterm( runningDevs, &baseProgs, config )

    // process devEvents
    event_loop( config, curIP, devEventCh, vpnEventCh, ifName, pubEventCh, runningDevs, &devMapLock, portMap, lineLog, &baseProgs, videoMethod )
}

func coordinator_NewLauncher( config *Config ) (*Launcher) {
    arguments := []string {
        "/Applications/STF Coordinator.app/Contents/MacOS/coordinator",
    }
    
    label     := fmt.Sprintf("com.tmobile.coordinator.app")
    wd        := "/Applications/STF Coordinator.app/Contents/MacOS"
    keepalive := true
    asRoot    := false
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
        devMapLock *sync.Mutex,
        portMap *PortMap,
        uuid string ) ( *RunningDev ) {
        
    wdaPort, vidPort, devIosPort, vncPort, usbmuxdPort, _, streamPort, config := assign_ports( gConfig, portMap )
  
    devd := RunningDev{
        uuid: uuid,
        shuttingDown:   false,
        failed:         false,
        wdaPort:        wdaPort,
        vidPort:        vidPort,
        devIosPort:     devIosPort,
        vncPort:        vncPort,
        usbmuxdPort:    usbmuxdPort,
        confDup:        config,
        videoReady:     false,
        streamWidth:    0,
        streamHeight:   0,
        streamPort:     streamPort,
        clickWidth:     0,
        clickHeight:    0,
        clickScale:     1000,
        okFirstFrame:   false,
        okVidInterface: false,
        wdaStarted:     false,
        process:        make( map[string] *GenericProc ),
        devUnitStarted: false,
        lock:           &sync.Mutex{},
        periodic:       make( chan bool ),
        wda:            false,
    }
    
    devd.name = getDeviceName( config, uuid )
    
    devd.iosVersion = getDeviceInfo( config, uuid, "ProductVersion" )
    
    devConf := get_device_config( config, uuid )
    if devConf != nil {
        devd.streamWidth = devConf.Width
        devd.streamHeight = devConf.Height
    }
    
    devMapLock.Lock()
    runningDevs[uuid] = &devd
    devMapLock.Unlock()
    
    log.WithFields( log.Fields{
        "type":         "devd_create",
        "dev_name":     devd.name,
        "dev_uuid":     censor_uuid( uuid ),
        "vid_port":     vidPort,
        "wda_port":     wdaPort,
        "vnc_port":     vncPort,
        "usbmuxd_port": usbmuxdPort,
        "dev_ios_port": devIosPort,
    } ).Info("Device object created")
    
    return &devd
}

func mini_event_loop( devEventCh <-chan DevEvent, devd *RunningDev ) {
    for {
        select {
        case devEvent := <- devEventCh:
            uuid := devd.uuid
            if devEvent.action == 3 { // first video frame
                devd.streamWidth  = devEvent.width
                devd.streamHeight = devEvent.height
                devd.clickScale   = devEvent.clickScale
                log.WithFields( log.Fields{
                    "type":       "first_frame",
                    "proc":       "ios_video_stream",
                    "width":      devEvent.width,
                    "height":     devEvent.height,
                    "clickScale": devEvent.clickScale,
                    "uuid":       censor_uuid( uuid ),
                } ).Info("Video - first frame")
            }
        }
    }
}

func event_loop(
        gConfig     *Config,
        curIP       string,
        devEventCh  chan DevEvent,
        vpnEventCh  <-chan VpnEvent,
        tunName     string,
        pubEventCh  chan<- PubEvent,
        runningDevs map[string] *RunningDev,
        devMapLock  *sync.Mutex,
        portMap     *PortMap,
        lineLog     *log.Entry,
        baseProgs   *BaseProgs,
        videoMethod string ) {
    
    gProcOptions := ProcOptions {
        config: gConfig,
        baseProgs: baseProgs,
        lineLog: lineLog,
    }

    if baseProgs.okStage1 == false {
        for {
            vpnEvent := <- vpnEventCh
            if vpnEvent.action == 0 {
                tunName = vpnEvent.text1
                curIP, _ = ifAddr( tunName )
                baseProgs.okVpn = true
            }
            
            if baseProgs.okVpn == true && baseProgs.okStage1 == false {
                proc_stf_provider( gProcOptions, curIP )
                baseProgs.okStage1 = true
                break
            }
        }
    }

    log.WithFields( log.Fields{
        "type":     "event_loop_start",
    } ).Debug("Event loop start")
    
    for {
        select {
        case vpnEvent := <- vpnEventCh:
            if vpnEvent.action == 0 {
                //iface := vpnEvent.text1
                // do nothing; assume tunnel interface is unchanged
            }
            
        case devEvent := <- devEventCh:
            if gStop { break }
            uuid := devEvent.uuid
    
            var devd *RunningDev = nil
            var ok = false
            
            o := ProcOptions {
                config:    gConfig,
                devd:      devd,
                baseProgs: baseProgs,
                lineLog:   lineLog,
                curIP:     curIP,
            }
            
            brandNew := false
            devMapLock.Lock()
            if devd, ok = runningDevs[uuid]; !ok {
                devMapLock.Unlock()
                devd = NewRunningDev( gConfig, runningDevs, devMapLock, portMap, uuid )
                periodic_start( gConfig, devd )
                brandNew = true
            } else {
                devMapLock.Unlock()
            }
            o.devd = devd
            
            if devEvent.action == 0 && brandNew == false {
                log.WithFields( log.Fields{
                    "type":     "dev_connect",
                    "dev_uuid": censor_uuid( uuid ),                
                } ).Info("Duplicate Device connect...")
            }
            
            if devEvent.action == 0 && brandNew == true { // device connect
                devName := devd.name
    
                log.WithFields( log.Fields{
                    "type":     "dev_connect",
                    "dev_name": devName,
                    "dev_uuid": censor_uuid( uuid ),                
                } ).Info("Device connected")
    
                if videoMethod == "ivp" {
                    ivp_enable( o )
                }
                
                o.config = devd.confDup
                
                if o.config.Video.Enabled {
                    if videoMethod == "avfoundation" {
                        proc_ios_video_stream( o, tunName, "127.0.0.1" )
                    	 proc_video_enabler( o )
                        proc_ivf( o )
                    } else if videoMethod == "ivp" {
                        proc_ios_video_stream( o, tunName, "127.0.0.1" )
                        proc_h264_to_jpeg( o )
                        proc_ios_video_pull( o )
                    } else if videoMethod == "app" {
                        proc_ios_video_stream( o, tunName, curIP )
                    }
                }
            }
            if devEvent.action == 1 { // device disconnect
                log.WithFields( log.Fields{
                    "type":     "dev_disconnect",
                    "dev_name": devd.name,
                    "dev_uuid": censor_uuid( uuid ),
                } ).Info("Device disconnected")
    
                periodic_stop( devd )
                closeRunningDev( devd, portMap )
    
                devMapLock.Lock()
                delete( runningDevs, uuid )
                devMapLock.Unlock()
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
                devd.okVidInterface = true
                log.WithFields( log.Fields{
                    "type":     "vid_available",
                    "dev_name": devd.name,
                    "dev_uuid": censor_uuid( uuid ),
                } ).Info("Video Interface Available")
            }
            if devEvent.action == 3 { // first video frame
                devd.okFirstFrame = true
                devd.streamWidth  = devEvent.width
                devd.streamHeight = devEvent.height
                devd.clickScale   = devEvent.clickScale
                log.WithFields( log.Fields{
                    "type":       "first_frame",
                    "proc":       "ios_video_stream",
                    "width":      devEvent.width,
                    "height":     devEvent.height,
                    "clickScale": devEvent.clickScale,
                    "uuid":       censor_uuid( uuid ),
                } ).Info("Video - first frame")
            }
            if devEvent.action == 4 { // WDA started
                devName := devd.name
                wdaPort := devd.wdaPort
                vidPort := devd.vidPort

                pubEventCh <- PubEvent{
                    action:  0, // connected
                    uuid:    uuid,
                    name:    devName,
                    wdaPort: wdaPort,
                    vidPort: vidPort,
                }
                
                pubEventCh <- PubEvent{
                    action: 3, // present
                    uuid: uuid,
                }              
              
                wdaBase := "http://127.0.0.1:" + strconv.Itoa( wdaPort )
                var sessionId string
                try := 0
                for {
                    resp, _ := http.Get( wdaBase + "/status" )
                    body := new(bytes.Buffer)
                    body.ReadFrom(resp.Body)
                    if string(body.Bytes()) != "" {
                        str := string(body.Bytes())
                        
                        str = strings.Replace( str, "true", "\"true\"", -1 )
                        str = strings.Replace( str, "false", "\"false\"", -1 )
                        fmt.Printf("Status response: %s\n", str )
                        root, _ := uj.Parse( []byte( str ) )
                        //root.Dump()
                        sessionNode := root.Get("sessionId")
                        if sessionNode == nil {
                            wda := NewWDACaller( wdaBase )
                            sessionId = wda.create_session( "com.apple.Preferences" )
                        } else {
                            sessionId = sessionNode.String()
                        }
                        
                        break
                    }
                    //fmt.Printf("trying again to getting wda session\n")
                    try++
                    if try > 36 {
                        break
                    }
                    time.Sleep( time.Second * 1 )
                }
                
                log.WithFields( log.Fields{
                    "type": "wda_session",
                    "id":   sessionId,
                    "uuid": censor_uuid( uuid ),
                } ).Info("Fetched WDA session")
                
                resp2, _ := http.Get( wdaBase + "/session/" + sessionId + "/window/size" )
                body2 := new(bytes.Buffer)
                body2.ReadFrom(resp2.Body)
                fmt.Printf("window size response: %s\n", string(body2.Bytes()) )
                root2, _ := uj.Parse( body2.Bytes() )
                
                val := root2.Get("value")
                devd.clickWidth  = val.Get("width").Int()
                devd.clickHeight = val.Get("height").Int()
                
                log.WithFields( log.Fields{
                    "type":  "device_dimensions",
                    "width":  devd.clickWidth,
                    "height": devd.clickHeight,
                    "uuid":   censor_uuid( uuid ),
                } ).Info("Fetched device screen dimensions")
                
                o.config = devd.confDup
                
                // Notify stf that the device is connected
                /*pubEvent := PubEvent{}
                pubEvent.action  = 0 // connected
                pubEvent.uuid    = devEvent.uuid
                pubEvent.name    = ""
                pubEvent.wdaPort = 0
                pubEvent.vidPort = 0
                pubEventCh <- pubEvent*/
                
                // start the heartbeat
                if devd.heartbeatChan == nil {
                    devd.heartbeatChan = coro_heartbeat( uuid, pubEventCh )
                }
                
                if videoMethod == "app" {
                    va_write_config( o.config, uuid, strconv.Itoa( o.config.DecodeInPort ), curIP )
                    
                    wda := NewWDACaller( wdaBase )
                    wda.launch_app( sessionId, "com.dryark.vidtest2" )
                    //sid := wda_session( wdaBase )
                    wda.start_broadcast( sessionId, "vidtest2" )
                }
                
                continue_dev_start( o, curIP )
            }
            if devEvent.action == 5 { // WDA Startup failed
                log.WithFields( log.Fields{
                    "type":     "wdaproxy_fail",
                    "dev_uuid": uuid,
                } ).Error("WDAProxy failed to start")
            }
                        
            if devd != nil && !devd.wdaStarted {
                o.config = devd.confDup
                
                if !o.config.Video.Enabled ||
                    ( o.devd.okVidInterface == true && o.devd.okFirstFrame == true ) ||
                    videoMethod == "app" {
                        o.devd.wdaStarted = true
                        
                        time.Sleep( time.Second * 2 )
                        
                        fmt.Printf("trying to get ios version\n")
                        
                        log.WithFields( log.Fields{
                            "type":        "ios_version",
                            "dev_name":    o.devd.name,
                            "dev_uuid":    uuid,
                            "ios_version": o.devd.iosVersion,
                        } ).Debug("IOS Version")
            
                        proc_wdaproxy( o, devEventCh, false )
                }
            }
        }
    }
}

func ivp_enable( o ProcOptions ) {
    uuid := o.devd.uuid
    devName := o.devd.name
    
    bytes, _ := exec.Command( "./bin/ios_video_pull", "-devices", "-json",
        "-udid", uuid,
        ).Output()
    root, _ := uj.Parse( bytes )
    activated := root.Get("activated").Int()
    
    if activated == 1 {
        log.WithFields( log.Fields{
            "type":     "dev_reset",
            "dev_name": devName,
            "dev_uuid": censor_uuid( uuid ),                
        } ).Info("Device already activated; resetting")
        
        /*
        // Reset the device
        time.Sleep( time.Second * 1 )
        exec.Command( "./bin/ios_video_pull", "-disable",
            "-udid", uuid ).Wait()
        */
        
        aio_reset_media_services( o )
        
        log.WithFields( log.Fields{
            "type":     "enabling",
            "dev_name": devName,
            "dev_uuid": censor_uuid( uuid ),                
        } ).Info("Device already activated; enabling after reset")
    }
        
    // Enable it
    time.Sleep( time.Second * 1 )
    exec.Command( "./bin/ios_video_pull", "-enable",
        "-udid", uuid ).Wait()
    
    time.Sleep( time.Second * 2 )
}

func censor_uuid( uuid string ) (string) {
    return "***" + uuid[len(uuid)-4:]
}

func continue_dev_start( o ProcOptions, curIP string ) {
    uuid := o.devd.uuid
    
    if o.config.Video.Enabled && o.config.Video.UseVnc {
        proc_vnc_proxy( o )
    }
    
    if !o.devd.devUnitStarted {
        o.devd.devUnitStarted = true
        proc_device_ios_unit( o, uuid, curIP )
    }
}
