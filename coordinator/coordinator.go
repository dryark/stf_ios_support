package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "io"
    "io/ioutil"
    "net"
    "net/http"
    "os"
    "os/exec"
    "os/signal"
    "strconv"
    "strings"
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
}

type BaseProgs struct {
    trigger    *os.Process
    vidEnabler *os.Process
    stf        *os.Process
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
    plog := log.WithFields( log.Fields{ "proc": "osx_ios_device_trigger" } )
    
    go func() {
        plog.WithFields( log.Fields{ "type": "proc_start" } ).Info("Starting: osx_ios_device_trigger")
        
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
        
        plog.WithFields( log.Fields{ "type": "proc_end" } ).Warn("Ended: osx_ios_device_trigger")
    }()
}

func proc_video_enabler( config *Config, baseProgs *BaseProgs ) {
    plog := log.WithFields( log.Fields{ "proc": "video_enabler" } )
    
    go func() {
        plog.WithFields( log.Fields{ "type": "proc_start" } ).Info("Starting: video-enabler")
        
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
        
        log.WithFields( log.Fields{ "type": "proc_end" } ).Warn("Ended: video-enabler")
    }()
}

func proc_stf( baseProgs *BaseProgs, tunName string ) {
    plog := log.WithFields( log.Fields{ "proc": "stf" } )
    
    go func() {
        for {
            plog.WithFields( log.Fields{
                "type": "proc_start",
                "tunnel_name": tunName,
            } ).Info("Starting: stf")
        
            stfCmd := exec.Command("/bin/bash", "run-stf.sh", tunName )
            stfCmd.Stdout = os.Stdout
            stfCmd.Stderr = os.Stderr
            
            err := stfCmd.Start()
            if err != nil {
                plog.WithFields( log.Fields{
                    "type": "proc_err",
                    "error": err,
                } ).Error("Error starting stf")
            
                baseProgs.stf = nil
            } else {
                baseProgs.stf = stfCmd.Process
            }
            stfCmd.Wait()
            
            plog.WithFields( log.Fields{ "type": "proc_end" } ).Warn("Ended: stf")
        }
    }()
}

func proc_mirrorfeed( config *Config, tunName string, devd *RunningDev ) ( string ) {
    plog := log.WithFields( log.Fields{ "proc": "mirrorfeed" } )
  
    mirrorPort := strconv.Itoa( config.MirrorFeedPort )
    mirrorFeedBin := config.MirrorFeedBin
    pipeName := config.Pipe
    
    plog.WithFields( log.Fields{
        "type": "proc_start",
        "mirrorfeed_bin": mirrorFeedBin,
        "pipe": pipeName,
        "port": mirrorPort,
    } ).Info("Starting: mirrorfeed")
    
    mirrorCmd := exec.Command( mirrorFeedBin, mirrorPort, pipeName, tunName )
    
    mirrorCmd.Stdout = os.Stdout
    mirrorCmd.Stderr = os.Stderr
    go func() {
        err := mirrorCmd.Start()
        if err != nil {
            plog.WithFields( log.Fields{
                "type": "proc_err",
                "error": err,
            } ).Error("Error starting mirrorfeed")
                
            devd.mirror = nil
        } else {
            devd.mirror = mirrorCmd.Process
        }
        mirrorCmd.Wait()
        devd.mirror = nil
        
        plog.WithFields( log.Fields{  "type": "proc_end" } ).Warn("Ended: mirrorfeed")
    }()
    
    return pipeName
}

func proc_ffmpeg( config *Config, devd *RunningDev, pipeName string, devName string ) {
    plog := log.WithFields( log.Fields{ "proc": "ffmpeg" } )
  
    plog.WithFields( log.Fields{ "type": "proc_start" } ).Info("Starting: ffmpeg")
                
    ffCmd := exec.Command( "./run-ffmpeg.sh", 
        config.Ffmpeg,
        pipeName,
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
    
    ffCmd.Stdout = os.Stdout
    ffCmd.Stderr = os.Stderr
    go func() {
        err := ffCmd.Start()
        if err != nil {
            plog.WithFields( log.Fields{
                "type": "proc_err",
                "error": err,
            } ).Error("Error starting ffmpeg")
            
            devd.ff = nil
        } else {
            devd.ff = ffCmd.Process
        }
        ffCmd.Wait()
        
        plog.WithFields( log.Fields{ "type": "proc_end" } ).Warn("Ended: ffmpeg")
        
        devd.ff = nil
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
    plog := log.WithFields( log.Fields{ "proc": "wdaproxy" } )

    // start wdaproxy
    wdaPort := config.WDAProxyPort
    
    cmd := fmt.Sprintf("wdaproxy -p %s -d -W %s -u %s", wdaPort, config.WDARoot, uuid )
    plog.WithFields( log.Fields{
      "type": "proc_start",
      "cmd": cmd,
    } ).Info("Starting wdaproxy")
    
    proxyCmd := exec.Command( "../../bin/wdaproxy", "-p", strconv.Itoa( wdaPort ), "-d", "-W", ".", "-u", uuid )

    proxyCmd.Stderr = os.Stderr
    proxyCmd.Dir = config.WDARoot
    go func() {
        proxyPipe, _ := proxyCmd.StdoutPipe()
        err := proxyCmd.Start()
        if err != nil {
            plog.WithFields( log.Fields{
                "type": "proc_err",
                "error": err,
            } ).Error("Error starting wdaproxy")
            
            devd.proxy = nil
        } else {
            devd.proxy = proxyCmd.Process
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
        
        wdaScanner := bufio.NewScanner( proxyPipe )
        for wdaScanner.Scan() {
            line := wdaScanner.Text()
            
            if strings.Contains( line, "is implemented in both" ) {
            } else if strings.Contains( line, "Couldn't write value" ) {
            } else {
                fmt.Println( line )
            }
        }
        
        stopChannel<- true
        
        plog.WithFields( log.Fields{ "type": "proc_end" } ).Warn("Ended: wdaproxy")
    }()
}

func proc_device_ios_unit( config *Config, devd *RunningDev, uuid string, curIP string ) {
    plog := log.WithFields( log.Fields{ "proc": "device_ios_unit" } )
        
    pushStr := fmt.Sprintf("tcp://%s:7270", config.STFIP)
    subStr := fmt.Sprintf("tcp://%s:7250", config.STFIP)
    
    cmd := fmt.Sprintf("./run-device-ios.sh --serial %s --connect-push %s --connect-sub %s --public-ip %s", pushStr, subStr, curIP )
    
    plog.WithFields( log.Fields{
      "type": "proc_start",
      "cmd": cmd,
    } ).Info("Starting wdaproxy")
    
    deviceCmd := exec.Command( "./run-device-ios.sh", "--serial", uuid, "--connect-push", pushStr, "--connect-sub", subStr, "--public-ip", curIP )
    deviceCmd.Stdout = os.Stdout
    deviceCmd.Stderr = os.Stderr
    go func() {
        err := deviceCmd.Start()
        if err != nil {
            plog.WithFields( log.Fields{
                "type": "proc_err",
                "error": err,
            } ).Error("Error starting device_ios_unit")
            
            devd.device = nil
        } else {
            devd.device = deviceCmd.Process
        }
        deviceCmd.Wait()
        devd.device = nil
        
        plog.WithFields( log.Fields{ "type": "proc_end" } ).Warn("Ended: device_ios_unit")
    }()
}

func main() {
    gStop = false
    
    // setup logging
    //log.SetFormatter( &log.JSONFormatter{} )
    log.SetLevel( log.InfoLevel )
    
    pubEventCh := make( chan PubEvent, 2 )
    
    if len( os.Args ) > 1 {
        handle_action()        
        return
    }
    
    coro_zmqReqRep()
    coro_zmqPub( pubEventCh )
    
    config := read_config()
    
    tunName, curIP, vpnMissing := get_net_info()

    cleanup_procs( config )
        
    devEventCh := make( chan DevEvent, 2 )
    runningDevs := make( map [string] RunningDev )
    baseProgs := BaseProgs{}
    
    coro_http_server( config, devEventCh )
    proc_device_trigger( config, &baseProgs )
    proc_video_enabler( config, &baseProgs )
    
    if vpnMissing {
        log.WithFields( log.Fields{ "type": "warn_vpn" } ).Warn("VPN not enabled; skipping start of STF")
        baseProgs.stf = nil
    } else {
        // start stf and restart it when needed
        // TODO: if it doesn't restart / crashes again; give up
        proc_stf( &baseProgs, tunName )
    }
	
    coro_CloseHandler( runningDevs, &baseProgs, config )
    
    // process devEvents
    event_loop( config, curIP, devEventCh, tunName, pubEventCh, runningDevs )
}

func handle_action() {
    action := os.Args[1]
    fmt.Printf("action: %s\n", action)
    
    if action == "server" {
        // nothing
    }
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

func event_loop( config *Config, curIP string, devEventCh <-chan DevEvent, tunName string, pubEventCh chan<- PubEvent, runningDevs map [string] RunningDev ) {
    for {
        // receive message
        devEvent := <- devEventCh
        uuid := devEvent.uuid
        
        if devEvent.action == 0 { // device connect
            devd := RunningDev{}
            devd.uuid = uuid
            
            devd.name = getDeviceName( uuid )
            devName := devd.name
            
            log.WithFields( log.Fields{
                "type": "dev_connect",
                "dev_name": devName,
                "dev_uuid": uuid,
            } ).Info("Device connected")
            
            if config.SkipVideo {
                devd.mirror = nil
                devd.ff = nil
            } else {
                pipeName := proc_mirrorfeed( config, tunName, &devd )
                proc_ffmpeg( config, &devd, pipeName, devName )
            
                // Sleep to ensure that video enabling process is finished before we try to start wdaproxy
                // This is needed because the USB device drops out and reappears during video enabling
                time.Sleep( time.Second * 9 )
            }
            
            proc_wdaproxy( config, &devd, &devEvent, uuid, devName, pubEventCh )
            
            time.Sleep( time.Second * 3 )
            
            proc_device_ios_unit( config, &devd, uuid, curIP )
            
            runningDevs[uuid] = devd
        }
        if devEvent.action == 1 { // device disconnect
            devd := runningDevs[uuid]
            closeRunningDev( devd )
            
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
        if cmdFlat == "/bin/bash run-stf.sh" {
            plog.WithFields( log.Fields{
                "proc": "stf",
            } ).Debug("Leftover Proc - Sending SIGTERM")
            
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
        if cmd[0] == "node" && cmd[1] == "--inspect=127.0.0.1:9230" {
            plog.WithFields( log.Fields{
                "proc": "stf_by_cmd,",
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
            }
            test := DevTest{}
            test.UUID = pubEvent.uuid
            test.Name = pubEvent.name
            
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

func closeAllRunningDevs( runningDevs map [string] RunningDev ) {
    for _, devd := range runningDevs {
        closeRunningDev( devd )
    }
}

func closeRunningDev( devd RunningDev ) {
    plog := log.WithFields( log.Fields{
        "type": "proc_cleanup_kill",
    } )
   
    if devd.proxy != nil {
        plog.WithFields( log.Fields{ "proc": "wdaproxy" } ).Debug("Killing proc")
        devd.proxy.Kill()
    }
    if devd.ff != nil {
        plog.WithFields( log.Fields{ "proc": "ffmpeg" } ).Debug("Killing proc")
        devd.ff.Kill()
    }
    if devd.mirror != nil {
        plog.WithFields( log.Fields{ "proc": "mirrorfeed" } ).Debug("Killing proc")
        devd.mirror.Kill()
    }
    if devd.device != nil {
        plog.WithFields( log.Fields{ "proc": "device_ios_unit" } ).Debug("Killing proc")
        devd.device.Kill()
    }
}

func closeBaseProgs( baseProgs *BaseProgs ) {
    plog := log.WithFields( log.Fields{
        "type": "proc_cleanup_kill",
    } )
  
    if baseProgs.trigger != nil {
        plog.WithFields( log.Fields{ "proc": "device_trigger" } ).Debug("Killing proc")
        baseProgs.trigger.Kill()
    }
    if baseProgs.vidEnabler != nil {
        plog.WithFields( log.Fields{ "proc": "video_enabler" } ).Debug("Killing proc")
        baseProgs.vidEnabler.Kill()
    }
    if baseProgs.stf != nil {
        plog.WithFields( log.Fields{ "proc": "stf" } ).Debug("Killing proc")
        baseProgs.stf.Kill()
    }
}

func coro_CloseHandler( runningDevs map [string] RunningDev, baseProgs *BaseProgs, config *Config ) {
    c := make(chan os.Signal, 2)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    go func() {
        <- c
        log.WithFields( log.Fields{
            "type": "shutdown",
            "state": "begun",
        } ).Info("Shutdown started")
                    
        closeBaseProgs( baseProgs )
        closeAllRunningDevs( runningDevs )
        
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
    name, _ := exec.Command( "idevicename", "-u", uuid ).Output()
    if name == nil || len(name) == 0 {
        fmt.Printf("idevicename returned nothing for uuid %s\n", uuid)
    }
    nameStr := string( name )
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