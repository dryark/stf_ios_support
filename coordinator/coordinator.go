package main

import (
    "bufio"
    "encoding/json"
    "fmt"
    "io"
    "io/ioutil"
    "log"
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
    
    zmq "github.com/zeromq/goczmq"
    ps "github.com/jviney/go-proc"
)

type Config struct {
    DeviceTrigger   string `json:"device_trigger"`
    VideoEnabler    string `json:"video_enabler"`
    SupportRoot     string `json:"support_root"`
    MirrorFeedBin   string `json:"mirrorfeed_bin"`
    WDARoot         string `json:"wda_root"`
    CoordinatorHost string `json:"coordinator_host"`
    CoordinatorPort int    `json:"coordinator_port"`
    WDAProxyBin     string `json:"wdaproxy_bin"`
    WDAProxyPort    string `json:"wdaproxy_port"`
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
        log.Panicf("failed reading file: %s", err)
    }
    defer configFh.Close()
      
    jsonBytes, _ := ioutil.ReadAll( configFh )
    var config Config
    json.Unmarshal( jsonBytes, &config )
    return &config
}

func proc_device_trigger( config *Config, baseProgs *BaseProgs ) {
    go func() {
        fmt.Printf("Starting osx_ios_device_trigger\n");
        triggerCmd := exec.Command( config.DeviceTrigger )
        
        err := triggerCmd.Start()
        if err != nil {
            fmt.Println(err.Error())
        } else {
            baseProgs.trigger = triggerCmd.Process
        }
        
        triggerCmd.Wait()
        fmt.Printf("Ended: osx_ios_device_trigger\n");
    }()
}

func proc_video_enabler( config *Config, baseProgs *BaseProgs ) {
    go func() {
        fmt.Printf("Starting video-enabler\n");
        enableCmd := exec.Command(config.VideoEnabler)
        err := enableCmd.Start()
        if err != nil {
            fmt.Println(err.Error())
            baseProgs.vidEnabler = nil
        } else {
            baseProgs.vidEnabler = enableCmd.Process 
        }
        enableCmd.Wait()
        fmt.Printf("Ended: video-enabler\n")
    }()
}

func proc_stf( baseProgs *BaseProgs, tunName string ) {
    go func() {
        for {
            fmt.Printf("Starting stf\n");
            stfCmd := exec.Command("/bin/bash", "run-stf.sh", tunName)
            stfCmd.Stdout = os.Stdout
            stfCmd.Stderr = os.Stderr
            
            err := stfCmd.Start()
            if err != nil {
                fmt.Println("Could not start stf")
                fmt.Println(err.Error())
                baseProgs.stf = nil
            } else {
                baseProgs.stf = stfCmd.Process
            }
            stfCmd.Wait()
            fmt.Printf("Ended:stf\n");
            // log out that it stopped
        }
    }()
}

func proc_mirrorfeed( config *Config, tunName string, devd *RunningDev ) ( string ) {
    mirrorPort := config.MirrorFeedPort
    pipeName := config.Pipe
    fmt.Printf("Starting mirrorfeed: %s\n", config.MirrorFeedBin);
    
    mirrorFeedBin := config.MirrorFeedBin
    
    mirrorCmd := exec.Command( mirrorFeedBin, strconv.Itoa( mirrorPort ), pipeName, tunName )
    
    mirrorCmd.Stdout = os.Stdout
    mirrorCmd.Stderr = os.Stderr
    go func() {
        err := mirrorCmd.Start()
        if err != nil {
            fmt.Println("Could not start mirrorfeed")
            fmt.Println(err.Error())
            devd.mirror = nil
        } else {
            devd.mirror = mirrorCmd.Process
        }
        mirrorCmd.Wait()
        fmt.Printf("mirrorfeed ended\n")
        devd.mirror = nil
    }()
    
    return pipeName
}

func proc_ffmpeg( config *Config, devd *RunningDev, pipeName string, devName string ) {
    fmt.Printf("Starting ffmpeg\n")
                
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
        "pipe:1" )
    
    ffCmd.Stdout = os.Stdout
    ffCmd.Stderr = os.Stderr
    go func() {
        err := ffCmd.Start()
        if err != nil {
            fmt.Println("Could not start ffmpeg")
            fmt.Println(err.Error())
            devd.ff = nil
        } else {
            devd.ff = ffCmd.Process
        }
        ffCmd.Wait()
        fmt.Printf("ffmpeg ended\n")
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
                pubEventCh <- beatEvent
            }
            time.Sleep( time.Second * 1 )
            count++;
        }
    }()
    
    return stopChannel
}

func proc_wdaproxy( config *Config, devd *RunningDev, devEvent *DevEvent, uuid string, devName string, pubEventCh chan<- PubEvent ) {
    // start wdaproxy
    wdaPort := config.WDAProxyPort // "8100"
    fmt.Printf("Starting wdaproxy\n")
    
    fmt.Printf("  wdaproxy -p %s -d -W %s -u %s\n", wdaPort, config.WDARoot, uuid )
    proxyCmd := exec.Command( "../../bin/wdaproxy", "-p", wdaPort, "-d", "-W", ".", "-u", uuid )

    proxyCmd.Stderr = os.Stderr
    proxyCmd.Dir = config.WDARoot
    go func() {
        proxyPipe, _ := proxyCmd.StdoutPipe()
        err := proxyCmd.Start()
        if err != nil {
            fmt.Println(err.Error())
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
        
        fmt.Printf("wdaproxy ended\n")
    }()
}

func proc_device_ios_unit( config *Config, devd *RunningDev, uuid string, curIP string ) {
    // Start the stf device-ios unit
    fmt.Printf("Starting device-ios unit\n")
    
    pushStr := fmt.Sprintf("tcp://%s:7270", config.STFIP)
    subStr := fmt.Sprintf("tcp://%s:7250", config.STFIP)
    fmt.Printf("  ./run-device-ios.sh --serial %s --connect-push %s --connect-sub %s --public-ip $s", pushStr, subStr )
    deviceCmd := exec.Command( "./run-device-ios.sh", "--serial", uuid, "--connect-push", pushStr, "--connect-sub", subStr, "--public-ip", curIP )
    deviceCmd.Stdout = os.Stdout
    deviceCmd.Stderr = os.Stderr
    go func() {
        err := deviceCmd.Start()
        if err != nil {
            fmt.Println("Could not start device-ios unit")
            fmt.Println(err.Error())
            devd.device = nil
        } else {
            devd.device = deviceCmd.Process
        }
        deviceCmd.Wait()
        fmt.Printf("device-ios unit ended\n")
        devd.device = nil
    }()
}

func main() {
    gStop = false
    
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
        fmt.Printf("VPN not enabled; skipping start of STF\n")
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
    fmt.Printf("Tunnel name: %s\n", tunName)
    if tunName != "none" {
        vpnMissing = false
    }
    curIP := ifaceCurIP( tunName )
    fmt.Printf("Current IP on VPN: %s\n", curIP)
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
            fmt.Printf("Setting up device uuid: %s\n", uuid)
            devd.name = getDeviceName( uuid )
            devName := devd.name
            fmt.Printf("Device name: %s\n", devName)
            
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
            pubEventCh <- pubEvent
        }
    }
}

func cleanup_procs(config *Config) {
    // Cleanup hanging processes if any
    procs := ps.GetAllProcessesInfo()
    for _, proc := range procs {
        cmd := proc.CommandLine
        //fmt.Printf("Proc: pid=%d %s\n", proc.Pid, proc.CommandLine )
        cmdFlat := strings.Join( cmd, " " )
        if cmdFlat == "/bin/bash run-stf.sh" {
            fmt.Printf("Leftover STF - Sending SIGTERM\n")
            syscall.Kill( proc.Pid, syscall.SIGTERM )
        }
        if cmdFlat == config.VideoEnabler {
            fmt.Printf("Leftover Video enabler - Sending SIGTERM\n")
            syscall.Kill( proc.Pid, syscall.SIGTERM )
        }
        if cmdFlat == config.DeviceTrigger {
            fmt.Printf("Leftover Device trigger - Sending SIGTERM\n")
            syscall.Kill( proc.Pid, syscall.SIGTERM )
        }
        if cmd[0] == "node" && cmd[1] == "--inspect=127.0.0.1:9230" {
            fmt.Printf("Leftover STF(via node) - Sending SIGTERM\n")
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
                    fmt.Printf("Unknown type\n")
            }
            fmt.Printf("Found an interface: %s\n", iface.Name )
            if iface.Name == tunName {
                fmt.Printf( "interface '%s' address: %s\n", tunName, ip.String() ) 
                ipStr = ip.String()
                foundInterface = true
            }
        }
    }
    if foundInterface == false {
        fmt.Printf( "Could not find interface %s\n", tunName )
        os.Exit( 1 )
    }
    
    return ipStr
}

func coro_zmqPub( pubEventCh <-chan PubEvent ) {
    var sentDummy bool = false
    
    // start the zmq pub mechanism
    go func() {
        pubSock := zmq.NewSock(zmq.Pub)
        defer pubSock.Destroy()
        
        _, err := pubSock.Bind("tcp://127.0.0.1:7294")
        if err != nil {
            log.Panicf("error binding: %s", err)
            os.Exit(1)
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
                log.Panicf("error encoding JSON: %s", err)
            }
            fmt.Printf("Publishing to stf: %s\n", reqMsg )
            
            pubSock.SendMessage( [][]byte{ []byte("devEvent"), reqMsg} )
        }
    }()
}

func coro_zmqReqRep() {
    go func() {
        repSock := zmq.NewSock(zmq.Rep)
        defer repSock.Destroy()
        
        _, err := repSock.Bind("tcp://127.0.0.1:7293")
        if err != nil {
            log.Panicf("error binding: %s", err)
            os.Exit(1)
        }
        
        repOb, err := zmq.NewReadWriter(repSock)
        if err != nil {
            log.Panicf("error making readwriter: %s", err)
            os.Exit(1)
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
                log.Panicf("error receiving: %s", err)
                os.Exit(1)
            }
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
                fmt.Printf("Received: %s\n", string( buf ) )
            
                response := []byte("response")
                repSock.SendMessage([][]byte{response})
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
    // stop wdaproxy
    if devd.proxy != nil {
        fmt.Printf("Killing wdaproxy\n")
        devd.proxy.Kill()
    }
    
    // stop ffmpeg
    if devd.ff != nil {
        fmt.Printf("Killing ffmpeg\n")
        devd.ff.Kill()
    }
    
    // stop mirrorfeed
    if devd.mirror != nil {
        fmt.Printf("Killing mirrorfeed\n")
        devd.mirror.Kill()
    }
    
    // stop device-ios unit
    if devd.device != nil {
        fmt.Printf("Killing device-ios unit\n")
        devd.device.Kill()
    }
}

func closeBaseProgs( baseProgs *BaseProgs ) {
    if baseProgs.trigger != nil {
        fmt.Printf("Killing trigger\n")
        baseProgs.trigger.Kill()
    }
    if baseProgs.vidEnabler != nil {
        fmt.Printf("Killing vidEnabler\n")
        baseProgs.vidEnabler.Kill()
    }
    if baseProgs.stf != nil {
        fmt.Printf("Killing stf\n")
        baseProgs.stf.Kill()
    }
}

func coro_CloseHandler( runningDevs map [string] RunningDev, baseProgs *BaseProgs, config *Config ) {
    c := make(chan os.Signal, 2)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    go func() {
        <- c
        fmt.Println("\nShutting down...\n")
        closeBaseProgs( baseProgs )
        closeAllRunningDevs( runningDevs )
        
        // This triggers zmq to stop receiving
        // We don't actually wait after this to ensure it has finished cleanly... oh well :)
        gStop = true
        
        time.Sleep( time.Millisecond * 1000 )
        cleanup_procs( config )
        fmt.Println("Shutdown ok\n")
        
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
    fmt.Printf("Starting server\n");
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