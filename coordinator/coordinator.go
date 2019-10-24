package main

import (
    "bufio"
    "encoding/json"
    "flag"
    "fmt"
    "io"
    "io/ioutil"
    "net"
    "net/http"
    "os"
    "os/exec"
    "os/signal"
    "sort"
    "strconv"
    "strings"
    "sync"
    "syscall"
    "time"
    "html/template"
    
    log "github.com/sirupsen/logrus"
    ps "github.com/jviney/go-proc"
    zmq "github.com/zeromq/goczmq"
)

type Config struct {
    DeviceTrigger   string `json:"device_trigger"`
    VideoEnabler    string `json:"video_enabler"`
    MirrorFeedBin   string `json:"mirrorfeed_bin"`
    WDARoot         string `json:"wda_root"`
    CoordinatorPort int    `json:"coordinator_port"`
    WDAProxyBin     string `json:"wdaproxy_bin"`
    WDAProxyPort    int    `json:"wdaproxy_port"`
    MirrorFeedPort  int    `json:"mirrorfeed_port"`
    Pipe            string `json:"pipe"`
    SkipVideo       bool   `json:"skip_video"`
    Ffmpeg          string `json:"ffmpeg"`
    STFIP           string `json:"stf_ip"`
    STFHostname     string `json:"stf_hostname"`
    WDAPorts        string `json:"wda_ports"`
    VidPorts        string `json:"vid_ports"`
}

type DevEvent struct {
    action int
    uuid string
}

type PubEvent struct {
    action int
    uuid string
    name string
    wdaPort int
    vidPort int
}

type RunningDev struct {
    uuid string
    name string
    mirror *os.Process
    ff     *os.Process
    proxy  *os.Process
    device *os.Process
    shuttingDown bool
    lock sync.Mutex
    failed bool
    wdaPort int
    vidPort int
}

type BaseProgs struct {
    trigger    *os.Process
    vidEnabler *os.Process
    stf        *os.Process
    shuttingDown bool
}

type PortItem struct {
    available bool
}

var gStop bool

func read_config() *Config {
    configFile := "config.json"
    configFh, err := os.Open(configFile)   
    if err != nil {
        log.WithFields( log.Fields{
            "type": "err_read_config",
            "config_file": configFile,
            "error": err,
        } ).Fatal("failed reading config file")
    }
    defer configFh.Close()
      
    jsonBytes, _ := ioutil.ReadAll( configFh )
    config := Config{
        DeviceTrigger: "bin/osx_ios_device_trigger",
        VideoEnabler: "bin/osx_ios_video_enabler",
        WDAProxyBin: "bin/wdaproxy",
        MirrorFeedBin: "bin/mirrorfeed",
        WDARoot: "./bin/wda",
        Ffmpeg: "bin/ffmpeg",
        CoordinatorPort: 8027,
        MirrorFeedPort: 8000,
        WDAProxyPort: 8100,
        Pipe: "pipe",
    }
    json.Unmarshal( jsonBytes, &config )
    return &config
}

func proc_device_trigger( config *Config, baseProgs *BaseProgs ) {
    plog := log.WithFields( log.Fields{ "proc": "device_trigger" } )
    
    go func() {
        plog.WithFields( log.Fields{ "type": "proc_start" } ).Info("Starting: device_trigger")
        
        triggerCmd := exec.Command( config.DeviceTrigger )
        
        err := triggerCmd.Start()
        if err != nil {
            plog.WithFields( log.Fields{
                "type": "proc_err",
                "error": err,
            } ).Error("Error starting device_trigger")
            
            baseProgs.trigger = nil
        } else {
            baseProgs.trigger = triggerCmd.Process
        }
        
        triggerCmd.Wait()
        
        plog.WithFields( log.Fields{ "type": "proc_end" } ).Warn("Ended: device_trigger")
    }()
}

func proc_video_enabler( config *Config, baseProgs *BaseProgs ) {
    plog := log.WithFields( log.Fields{ "proc": "video_enabler" } )
    
    go func() {
        plog.WithFields( log.Fields{ "type": "proc_start" } ).Info("Starting: video_enabler")
        
        enableCmd := exec.Command(config.VideoEnabler)
        err := enableCmd.Start()
        if err != nil {
            plog.WithFields( log.Fields{
                "type": "proc_err",
                "error": err,
            } ).Error("Error starting video_enabler")
            
            baseProgs.vidEnabler = nil
        } else {
            baseProgs.vidEnabler = enableCmd.Process 
        }
        enableCmd.Wait()
        
        plog.WithFields( log.Fields{ "type": "proc_end" } ).Warn("Ended: video_enabler")
    }()
}

func proc_stf_provider( baseProgs *BaseProgs, curIP string, config *Config ) {
    plog := log.WithFields( log.Fields{ "proc": "stf_provider" } )
    
    go func() {
        for {
            serverHostname := config.STFHostname
            clientHostname, _ := os.Hostname()
            serverIP := config.STFIP
            
            plog.WithFields( log.Fields{
                "type": "proc_start",
                "client_ip": curIP,
                "server_ip": serverIP,
                "client_hostname": clientHostname,
                "server_hostname": serverHostname,
            } ).Info("Starting: stf_provider")
        
            cmd := exec.Command( "node", "--inspect=127.0.0.1:9230", "runmod.js", "provider",
                "--name"         , fmt.Sprintf("macmini/%s", clientHostname),
                "--connect-sub"  , fmt.Sprintf("tcp://%s:7250", serverIP),
                "--connect-push" , fmt.Sprintf("tcp://%s:7270", serverIP),
                "--storage-url"  , fmt.Sprintf("https://%s/", serverHostname),
                "--public-ip"    , curIP,
                "--min-port=7400",
                "--max-port=7700",
                "--heartbeat-interval=10000",
                "--server-ip"    , serverIP,
                "--no-cleanup" )
            
            outputPipe, _ := cmd.StderrPipe()
            cmd.Dir = "./repos/stf"
            cmd.Stdout = os.Stdout
            
            err := cmd.Start()
            if err != nil {
                plog.WithFields( log.Fields{
                    "type": "proc_err",
                    "error": err,
                } ).Error("Error starting stf")
            
                baseProgs.stf = nil
            } else {
                baseProgs.stf = cmd.Process
            }
            
            scanner := bufio.NewScanner( outputPipe )
            for scanner.Scan() {
                line := scanner.Text()
                if strings.Contains( line, " IOS Heartbeat:" ) {
                } else {
                    fmt.Printf( "[PROVIDER] %s\n", line )
                }
            }
            
            plog.WithFields( log.Fields{ "type": "proc_end" } ).Warn("Ended: stf_provider")
            
            if baseProgs.shuttingDown {
                break
            }
            
            // sleep before restart to prevent rapid failing attempts
            time.Sleep( time.Second * 5 )
        }
    }()
}

func proc_mirrorfeed( config *Config, tunName string, devd *RunningDev ) {
    plog := log.WithFields( log.Fields{ "proc": "mirrorfeed" } )
  
    mirrorPort := strconv.Itoa( config.MirrorFeedPort )
    mirrorFeedBin := config.MirrorFeedBin
    pipeName := config.Pipe
    
    if devd.shuttingDown {
        return
    }
    go func() {
        for {
            plog.WithFields( log.Fields{
                "type": "proc_start",
                "mirrorfeed_bin": mirrorFeedBin,
                "pipe": pipeName,
                "port": mirrorPort,
            } ).Info("Starting: mirrorfeed")
            
            cmd := exec.Command( mirrorFeedBin, mirrorPort, pipeName, tunName )
            
            outputPipe, _ := cmd.StdoutPipe()
            cmd.Stderr = os.Stderr
            
            err := cmd.Start()
            if err != nil {
                plog.WithFields( log.Fields{
                    "type": "proc_err",
                    "error": err,
                } ).Error("Error starting mirrorfeed")
                    
                devd.mirror = nil
            } else {
                devd.mirror = cmd.Process
            }
            
            scanner := bufio.NewScanner( outputPipe )
            for scanner.Scan() {
                line := scanner.Text()
                fmt.Printf( "[VIDFEED-] %s\n", line )
            }
            
            devd.mirror = nil
            
            plog.WithFields( log.Fields{  "type": "proc_end" } ).Warn("Ended: mirrorfeed")
            
            devd.lock.Lock()
            exit := devd.shuttingDown
            devd.lock.Unlock()
            if exit { break }
        }
    }()
}

func proc_ffmpeg( config *Config, devd *RunningDev, devName string ) {
    plog := log.WithFields( log.Fields{ "proc": "ffmpeg" } )
     
    if devd.shuttingDown {
        return
    }
    go func() {
        for {
            plog.WithFields( log.Fields{ "type": "proc_start" } ).Info("Starting: ffmpeg")
          
            cmd := exec.Command( "./run-ffmpeg.sh", 
                config.Ffmpeg,
                config.Pipe,
                devName,
                "-pixel_format", "bgr0",
                "-f", "mjpeg",
                "-bsf:v", "mjpegadump",
                "-bsf:v", "mjpeg2jpeg",
                "-r", "1", // framerate
                "-vsync", "2",
                "-nostats",
                // "-progress", "[url]",
                "pipe:1" )
            
            outputPipe, _ := cmd.StderrPipe()
            cmd.Stdout = os.Stdout
            
            err := cmd.Start()
            if err != nil {
                plog.WithFields( log.Fields{
                    "type": "proc_err",
                    "error": err,
                } ).Error("Error starting ffmpeg")
                
                devd.ff = nil
            } else {
                devd.ff = cmd.Process
            }
            
            scanner := bufio.NewScanner( outputPipe )
            for scanner.Scan() {
                line := scanner.Text()
                
                if strings.Contains( line, "avfoundation @" ) {
                } else if strings.Contains( line, "swscaler @" ) {
                } else if strings.HasPrefix( line, "  lib" ) {
                } else {
                  fmt.Printf( "[FFMPEG--] %s\n", line )
                }
            }
            
            plog.WithFields( log.Fields{ "type": "proc_end" } ).Warn("Ended: ffmpeg")
            
            devd.ff = nil
            
            devd.lock.Lock()
            exit := devd.shuttingDown
            devd.lock.Unlock()
            if exit { break }
            
            // sleep before restart to prevent rapid failing attempts
            time.Sleep( time.Second * 5 )
        }
    }()
}

func coro_heartbeat( devEvent *DevEvent, pubEventCh chan<- PubEvent ) ( chan<- bool ) {
    count := 1
    stopChannel := make(chan bool)
        
    // Start heartbeat
    go func() {
        done := false
        for {
            select {
                case _ = <-stopChannel:
                    done = true
                default:
            }
            if done {
                break
            }

            if count >= 10 {
                count = 0
                
                beatEvent := PubEvent{}
                beatEvent.action = 2
                beatEvent.uuid = devEvent.uuid
                beatEvent.name = ""
                beatEvent.wdaPort = 0
                beatEvent.vidPort = 0
                pubEventCh <- beatEvent
            }
            time.Sleep( time.Second * 1 )
            count++;
        }
    }()
    
    return stopChannel
}

func proc_wdaproxy( config *Config, devd *RunningDev, devEvent *DevEvent, uuid string, devName string, pubEventCh chan<- PubEvent ) {
    plog := log.WithFields( log.Fields{
      "proc": "wdaproxy",
      "uuid": devd.uuid,
    } )

    // start wdaproxy
    wdaPort := config.WDAProxyPort
    
    cmd := fmt.Sprintf("wdaproxy -p %d -d -W %s -u %s", wdaPort, config.WDARoot, uuid )
    
    if devd.shuttingDown {
        return
    }
    go func() {
        for {
            plog.WithFields( log.Fields{
              "type": "proc_start",
              "cmd": cmd,
            } ).Info("Starting wdaproxy")
            
            cmd := exec.Command( "../../bin/wdaproxy", "-p", strconv.Itoa( wdaPort ), "-d", "-W", ".", "-u", uuid )
        
            //cmd.Stderr = os.Stderr
            cmd.Dir = config.WDARoot
        
            outputPipe, _ := cmd.StdoutPipe()
            errPipe, _ := cmd.StderrPipe()
            err := cmd.Start()
            if err != nil {
                plog.WithFields( log.Fields{
                    "type": "proc_err",
                    "error": err,
                } ).Error("Error starting wdaproxy")
                
                devd.proxy = nil
            } else {
                devd.proxy = cmd.Process
            }
            
            time.Sleep( time.Second * 3 )
            
            // Everything is started; notify stf via zmq published event
            pubEvent := PubEvent{}
            pubEvent.action = devEvent.action
            pubEvent.uuid = devEvent.uuid
            pubEvent.name = devName
            pubEvent.wdaPort = config.WDAProxyPort
            pubEvent.vidPort = config.MirrorFeedPort
            pubEventCh <- pubEvent
            
            stopChannel := coro_heartbeat( devEvent, pubEventCh )
            
            go func() {
                scanner := bufio.NewScanner( outputPipe )
                for scanner.Scan() {
                    line := scanner.Text()
                    
                    if strings.Contains( line, "is implemented in both" ) {
                    } else if strings.Contains( line, "Couldn't write value" ) {
                    } else if strings.Contains( line, "GET /status " ) {
                    } else if strings.Contains( line, "[WDA] successfully started" ) {
                        plog.WithFields( log.Fields{ "type": "wda_started" } ).Info("WDA started")
                        fmt.Printf( "[WDAPROXY] %s\n", line )
                    } else {
                        fmt.Printf( "[WDAPROXY] %s\n", line )
                    }
                }
            } ()
            scanner := bufio.NewScanner( errPipe )
            for scanner.Scan() {
                line := scanner.Text()
                
                if strings.Contains( line, "[WDA] successfully started" ) {
                    plog.WithFields( log.Fields{ "type": "wda_started" } ).Info("WDA started")
                    fmt.Printf( "[WDAPROXE] %s\n", line )
                } else {
                    fmt.Printf( "[WDAPROXE] %s\n", line )
                }
            }
            
            stopChannel<- true
            
            plog.WithFields( log.Fields{ "type": "proc_end" } ).Warn("Ended: wdaproxy")
            
            devd.lock.Lock()
            exit := devd.shuttingDown
            devd.lock.Unlock()
            if exit { break }
        }
    }()
}

func proc_device_ios_unit( config *Config, devd *RunningDev, uuid string, curIP string ) {
    plog := log.WithFields( log.Fields{
      "proc": "stf_device_ios",
      "uuid": uuid,
    } )
        
    pushStr := fmt.Sprintf("tcp://%s:7270", config.STFIP)
    subStr := fmt.Sprintf("tcp://%s:7250", config.STFIP)
    
    go func() {
        for {
            plog.WithFields( log.Fields{
              "type": "proc_start",
              "server_ip": config.STFIP,
              "client_ip": curIP,
            } ).Info("Starting stf_device_ios")
            
            cmd := exec.Command( "node", "runmod.js", "device-ios",
                "--serial", uuid,
                "--connect-push", pushStr,
                "--connect-sub", subStr,
                "--public-ip", curIP,
                "--wda-port", strconv.Itoa( config.WDAProxyPort ), 
                //"--vid-port", strconv.Itoa( config.MirrorFeedPort ),
            )
            cmd.Dir = "./repos/stf"
            outputPipe, _ := cmd.StderrPipe()
            cmd.Stdout = os.Stdout
            
            err := cmd.Start()
            if err != nil {
                plog.WithFields( log.Fields{
                    "type": "proc_err",
                    "error": err,
                } ).Error("Error starting device_ios_unit")
                
                devd.device = nil
            } else {
                devd.device = cmd.Process
            }
            
            scanner := bufio.NewScanner( outputPipe )
            for scanner.Scan() {
                line := scanner.Text()
                fmt.Printf( "[StfDvIos] %s\n", line )
            }
            
            devd.device = nil
            
            plog.WithFields( log.Fields{ "type": "proc_end" } ).Warn("Ended: stf_device_ios")
            
            devd.lock.Lock()
            exit := devd.shuttingDown
            devd.lock.Unlock()
            if exit { break }
        }
    }()
}

func main() {
    gStop = false
    
    var debug = flag.Bool( "debug", false, "Use debug log level" )
    var jsonLog = flag.Bool( "json", false, "Use json log output" )
    flag.Parse()
    
    if *jsonLog {
        log.SetFormatter( &log.JSONFormatter{} )
    }
    
    if *debug {
        log.WithFields( log.Fields{ "type": "debug_status" } ).Warn("Debugging enabled")
        log.SetLevel( log.DebugLevel )
    } else {
        log.SetLevel( log.InfoLevel )
    }
    
    pubEventCh := make( chan PubEvent, 2 )
    
    coro_zmqReqRep()
    coro_zmqPub( pubEventCh )
    
    config := read_config()
    
    tunName, curIP, vpnMissing := get_net_info()

    cleanup_procs( config )
        
    devEventCh := make( chan DevEvent, 2 )
    runningDevs := make( map [string] *RunningDev )
    wdaPorts := construct_ports( config, config.WDAPorts )
    vidPorts := construct_ports( config, config.VidPorts )
    baseProgs := BaseProgs{}
    baseProgs.shuttingDown = false
    
    coro_http_server( config, devEventCh )
    proc_device_trigger( config, &baseProgs )
    proc_video_enabler( config, &baseProgs )
    
    if vpnMissing {
        log.WithFields( log.Fields{ "type": "warn_vpn" } ).Warn("VPN not enabled; skipping start of STF")
        baseProgs.stf = nil
    } else {
        // start stf and restart it when needed
        // TODO: if it doesn't restart / crashes again; give up
        proc_stf_provider( &baseProgs, curIP, config )
    }
	
    coro_CloseHandler( runningDevs, &baseProgs, config )
    
    // process devEvents
    event_loop( config, curIP, devEventCh, tunName, pubEventCh, runningDevs, wdaPorts, vidPorts )
}

func construct_ports( config *Config, spec string ) ( map [int] *PortItem ) {
    ports := make( map [int] *PortItem )
    if strings.Contains( spec, "-" ) {
        parts := strings.Split( spec, "-" )
        from, _ := strconv.Atoi( parts[0] )
        to, _ := strconv.Atoi( parts[1] )
        for i := from; i <= to; i++ {
            portItem := PortItem{
                available: true,
            }
            ports[ i ] = &portItem
        }
    }
    return ports
}

func coro_http_server( config *Config, devEventCh chan<- DevEvent ) {
    // start web server waiting for trigger http command for device connect and disconnect
    var listen_addr = fmt.Sprintf( "0.0.0.0:%d", config.CoordinatorPort )
    go startServer( devEventCh, listen_addr )
}

func get_net_info() ( string, string, bool ) {
    var vpnMissing bool = true
    tunName := getTunName()
    
    log.WithFields( log.Fields{
        "type": "info_vpn",
        "tunnel_name": tunName,
    } ).Info("Tunnel name")
    
    if tunName != "none" {
        vpnMissing = false
    }
    curIP := ifaceCurIP( tunName )
    
    log.WithFields( log.Fields{
        "type": "info_vpn",
        "tunnel_name": tunName,
        "ip": curIP,
    } ).Info("IP on VPN")
    
    return tunName, curIP, vpnMissing
}

func assign_ports( gConfig *Config, wdaPorts map [int] *PortItem, vidPorts map [int] *PortItem ) ( int,int,*Config ) {
    dupConfig := *gConfig
    
    wdaPort := 0
    vidPort := 0
    
    wKeys := make( []int, len(wdaPorts) )
    wI := 0
    for k := range wdaPorts {
        //fmt.Printf("a w port %d\n", k)
        wKeys[wI] = k
        wI++
    }
    sort.Ints( wKeys )
    
    vKeys := make( []int, len(vidPorts) )
    vI := 0
    for k := range vidPorts {
        //fmt.Printf("a v port %d\n", k)
        vKeys[vI] = k
        vI++
    }
    sort.Ints( vKeys )
    
    for _,port := range wKeys {
        //fmt.Printf("w port %d\n", port)
        portItem := wdaPorts[port]
        if portItem.available {
            portItem.available = false
            dupConfig.WDAProxyPort = port
            wdaPort = port
            break
        }
    }
    
    for _,port := range vKeys {
        //fmt.Printf("v port %d\n", port)
        portItem := vidPorts[port]
        if portItem.available {
            portItem.available = false
            dupConfig.MirrorFeedPort = port
            vidPort = port
            break
        }
    }
    
    return wdaPort, vidPort, &dupConfig
}

func free_ports( wdaPort int, vidPort int, wdaPorts map [int] *PortItem, vidPorts map [int] *PortItem ) {
    wdaItem := wdaPorts[ wdaPort ]
    wdaItem.available = true
    
    vidItem := vidPorts[ vidPort ]
    vidItem.available = true    
}

func event_loop( 
        gConfig *Config,
        curIP string,
        devEventCh <-chan DevEvent,
        tunName string,
        pubEventCh chan<- PubEvent,
        runningDevs map [string] *RunningDev,
        wdaPorts map [int] *PortItem,
        vidPorts map [int] *PortItem ) {
    for {
        // receive message
        devEvent := <- devEventCh
        uuid := devEvent.uuid
        
        if devEvent.action == 0 { // device connect
            wdaPort, vidPort, config := assign_ports( gConfig, wdaPorts, vidPorts )
            
            devd := RunningDev{
                uuid: uuid,
                shuttingDown: false,
                failed: false,
                mirror: nil,
                ff: nil,
                device: nil,
                proxy: nil,
                wdaPort: wdaPort,
                vidPort: vidPort,
            }
            runningDevs[uuid] = &devd
            
            devd.name = getDeviceName( uuid )
            if devd.name == "" {
                devd.failed = true
                continue
            }
            devName := devd.name
            
            log.WithFields( log.Fields{
                "type": "dev_connect",
                "dev_name": devName,
                "dev_uuid": uuid,
                "vid_port": vidPort,
                "wda_port": wdaPort,
            } ).Info("Device connected")
            
            if !config.SkipVideo {
                proc_mirrorfeed( config, tunName, &devd )
                proc_ffmpeg( config, &devd, devName )
            
                // Sleep to ensure that video enabling process is finished before we try to start wdaproxy
                // This is needed because the USB device drops out and reappears during video enabling
                time.Sleep( time.Second * 9 )
            }
            
            proc_wdaproxy( config, &devd, &devEvent, uuid, devName, pubEventCh )
            
            time.Sleep( time.Second * 3 )
            
            proc_device_ios_unit( config, &devd, uuid, curIP )
        }
        if devEvent.action == 1 { // device disconnect
            devd := runningDevs[uuid]
            
            log.WithFields( log.Fields{
                "type": "dev_disconnect",
                "dev_name": devd.name,
                "dev_uuid": uuid,
            } ).Info("Device disconnected")
            
            closeRunningDev( devd, wdaPorts, vidPorts )
            
            // Notify stf that the device is gone
            pubEvent := PubEvent{}
            pubEvent.action = devEvent.action
            pubEvent.uuid = devEvent.uuid
            pubEvent.name = ""
            pubEvent.wdaPort = 0
            pubEvent.vidPort = 0
            pubEventCh <- pubEvent
        }
    }
}

func cleanup_procs(config *Config) {
    plog := log.WithFields( log.Fields{
        "type": "proc_cleanup_kill",
    } )
  
    // Cleanup hanging processes if any
    procs := ps.GetAllProcessesInfo()
    for _, proc := range procs {
        cmd := proc.CommandLine
        cmdFlat := strings.Join( cmd, " " )
        if cmd[0] == "bin/mirrorfeed" {
            plog.WithFields( log.Fields{
                "proc": "mirrorfeed",
            } ).Debug("Leftover Mirrorfeed - Sending SIGTERM")
            
            syscall.Kill( proc.Pid, syscall.SIGTERM )
        }
        if cmdFlat == config.VideoEnabler {
            plog.WithFields( log.Fields{
                "proc": "video_enabler",
            } ).Debug("Leftover Proc - Sending SIGTERM")
            
            syscall.Kill( proc.Pid, syscall.SIGTERM )
        }
        if cmdFlat == config.DeviceTrigger {
            plog.WithFields( log.Fields{
                "proc": "device_trigger",
            } ).Debug("Leftover Proc - Sending SIGTERM")
            
            syscall.Kill( proc.Pid, syscall.SIGTERM )
        }
        
        // node runmod.js device-ios
        if cmd[0] == "node" && cmd[2] == "device-ios" {
            plog.WithFields( log.Fields{
                "proc": "device-ios",
            } ).Debug("Leftover Proc - Sending SIGTERM")
            
            syscall.Kill( proc.Pid, syscall.SIGTERM )
        }
        
        // node --inspect=[ip]:[port] runmod.js provider
        if cmd[0] == "node" && cmd[3] == "provider" {
            plog.WithFields( log.Fields{
                "proc": "stf_provider",
            } ).Debug("Leftover Proc - Sending SIGTERM")
            
            syscall.Kill( proc.Pid, syscall.SIGTERM )
        }
    }
}

func getTunName() string {
    i := 0
    iface := "none"
    for {
        cmd := fmt.Sprintf( "echo show State:/Network/Interface/utun%d/IPv4 | scutil | grep ' Addresses' -A 1 | tail -1 | awk '{print $3;}'", i )
        out, _ := exec.Command( "bash", "-c", cmd ).Output()
        if len( out ) > 0 {
            iface = fmt.Sprintf( "utun%d", i )
            break
        }
        i++
        if i > 1 {
            break
        }
    }

    return iface
}

func ifaceCurIP( tunName string ) string {
    ifaces, err := net.Interfaces()
    if err != nil {
        fmt.Printf( err.Error() )
        os.Exit( 1 )
    }
    
    ipStr := ""
    
    foundInterface := false
    for _, iface := range ifaces {
        addrs, err := iface.Addrs()
        if err != nil {
            fmt.Printf( err.Error() )
            os.Exit( 1 )
        }
        for _, addr := range addrs {
            var ip net.IP
            switch v := addr.(type) {
                case *net.IPNet:
                    ip = v.IP
                case *net.IPAddr:
                    ip = v.IP
                default:
            }
            
            log.WithFields( log.Fields{
                "type": "net_interface_found",
                "interface_name": iface.Name,
            } ).Debug("Found an interface")
            
            if iface.Name == tunName {
                log.WithFields( log.Fields{
                    "type": "net_interface_info",
                    "interface_name": tunName,
                    "ip": ip.String(),
                } ).Debug("Interface Details")
            
                ipStr = ip.String()
                foundInterface = true
            }
        }
    }
    if foundInterface == false {
        log.WithFields( log.Fields{
            "type": "err_net_interface",
            "interface_name": tunName,
        } ).Fatal("Could not find interface")
    }
    
    return ipStr
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
                Type string
                UUID string
                Name string
                VidPort string
                WDAPort string
            }
            test := DevTest{}
            test.UUID = pubEvent.uuid
            test.Name = pubEvent.name
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

func closeAllRunningDevs( runningDevs map [string] *RunningDev ) {
    for _, devd := range runningDevs {
        closeRunningDev( devd, nil, nil )
    }
}

func closeRunningDev( devd *RunningDev, wdaPorts map [int] *PortItem, vidPorts map [int] *PortItem ) {
    devd.lock.Lock()
    devd.shuttingDown = true
    devd.lock.Unlock()
    
    if wdaPorts != nil && vidPorts != nil {
        free_ports( devd.wdaPort, devd.vidPort, wdaPorts, vidPorts )
    }
    
    plog := log.WithFields( log.Fields{
        "type": "proc_cleanup_kill",
        "uuid": devd.uuid,
    } )
    
    plog.Info("Closing running dev")
   
    if devd.proxy != nil {
        plog.WithFields( log.Fields{ "proc": "wdaproxy" } ).Debug("Killing wdaproxy")
        devd.proxy.Kill()
    }
    if devd.ff != nil {
        plog.WithFields( log.Fields{ "proc": "ffmpeg" } ).Debug("Killing ffmpeg")
        devd.ff.Kill()
    }
    if devd.mirror != nil {
        plog.WithFields( log.Fields{ "proc": "mirrorfeed" } ).Debug("Killing mirrorfeed")
        devd.mirror.Kill()
    }
    if devd.device != nil {
        plog.WithFields( log.Fields{ "proc": "device_ios_unit" } ).Debug("Killing device_ios_unit")
        devd.device.Kill()
    }
}

func closeBaseProgs( baseProgs *BaseProgs ) {
    baseProgs.shuttingDown = true
    
    plog := log.WithFields( log.Fields{
        "type": "proc_cleanup_kill",
    } )
  
    if baseProgs.trigger != nil {
        plog.WithFields( log.Fields{ "proc": "device_trigger" } ).Debug("Killing device_trigger")
        baseProgs.trigger.Kill()
    }
    if baseProgs.vidEnabler != nil {
        plog.WithFields( log.Fields{ "proc": "video_enabler" } ).Debug("Killing video_enabler")
        baseProgs.vidEnabler.Kill()
    }
    if baseProgs.stf != nil {
        plog.WithFields( log.Fields{ "proc": "stf_provider" } ).Debug("Killing stf_provider")
        baseProgs.stf.Kill()
    }
}

func coro_CloseHandler( runningDevs map [string] *RunningDev, baseProgs *BaseProgs, config *Config ) {
    c := make(chan os.Signal, 2)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    go func() {
        <- c
        log.WithFields( log.Fields{
            "type": "shutdown",
            "state": "begun",
        } ).Info("Shutdown started")
        
        closeAllRunningDevs( runningDevs )
        closeBaseProgs( baseProgs )
          
        // This triggers zmq to stop receiving
        // We don't actually wait after this to ensure it has finished cleanly... oh well :)
        gStop = true
        
        time.Sleep( time.Millisecond * 1000 )
        cleanup_procs( config )
        
        log.WithFields( log.Fields{
            "type": "shutdown",
            "state": "done",
        } ).Info("Shutdown finished")
        
        os.Exit(0)
    }()
}

func getDeviceName( uuid string ) (string) {
    i := 0
    var nameStr string
    for {
        i++
        if i > 10 { return "" }
        name, _ := exec.Command( "idevicename", "-u", uuid ).Output()
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
	
func startServer( devEventCh chan<- DevEvent, listen_addr string ) {
    log.WithFields( log.Fields{
        "type": "http_start",
    } ).Info("HTTP started")
        
    http.HandleFunc( "/", handleRoot )
    connectClosure := func( w http.ResponseWriter, r *http.Request ) {
        deviceConnect( w, r, devEventCh )
    }
    disconnectClosure := func( w http.ResponseWriter, r *http.Request ) {
        deviceDisconnect( w, r, devEventCh )
    }
    http.HandleFunc( "/dev_connect", connectClosure )
    http.HandleFunc( "/dev_disconnect", disconnectClosure )
    log.Fatal( http.ListenAndServe( listen_addr, nil ) )
}

func handleRoot( w http.ResponseWriter, r *http.Request ) {
    rootTpl.Execute( w, "ws://"+r.Host+"/echo" )
}

func fixUuid( uuid string ) (string) {
    if len(uuid) == 24 {
        p1 := uuid[:8]
        p2 := uuid[8:]
        uuid = fmt.Sprintf("%s-%s",p1,p2)
    }
    return uuid
}

func deviceConnect( w http.ResponseWriter, r *http.Request, devEventCh chan<- DevEvent ) {
    // signal device loop of device connect
    devEvent := DevEvent{}
    devEvent.action = 0
    r.ParseForm()
    uuid := r.Form.Get("uuid")
    uuid = fixUuid( uuid )
    devEvent.uuid = uuid	
    devEventCh <- devEvent
}

func deviceDisconnect( w http.ResponseWriter, r *http.Request, devEventCh chan<- DevEvent ) {
    // signal device loop of device disconnect
    devEvent := DevEvent{}
    devEvent.action = 1
    r.ParseForm()
    uuid := r.Form.Get("uuid")
    uuid = fixUuid( uuid )
    devEvent.uuid = uuid
    devEventCh <- devEvent
}

var rootTpl = template.Must(template.New("").Parse(`
<!DOCTYPE html>
<html>
	<head>
	</head>
	<body>
	test
	</body>
</html>
`))