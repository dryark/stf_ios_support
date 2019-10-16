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
	//ps "github.com/mitchellh/go-ps"
	ps "github.com/jviney/go-proc"
)

type Config struct {
	DeviceTrigger string `json:"device_trigger"`
	VideoEnabler string `json:"video_enabler"`
	SupportRoot string `json:"support_root"`
	MirrorFeedBin string `json:"mirrorfeed_bin"`
	WDARoot string `json:"wda_root"`
	CoordinatorHost string `json:"coordinator_host"`
	CoordinatorPort int `json:"coordinator_port"`
	WDAProxyBin string `json:"wdaproxy_bin"`
	WDAProxyPort string `json:"wdaproxy_port"`
	MirrorFeedPort int `json:"mirrorfeed_port"`
	Pipe string `json:"pipe"`
	SkipVideo bool `json:"skip_video"`
	Ffmpeg string `json:"ffmpeg"`
	STFIP string `json:"stf_ip"`
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

func main() {
    gStop = false
    
    pubEventCh := make( chan PubEvent, 2 )
    
    if len( os.Args ) > 1 {
        arg := os.Args[1]
        fmt.Printf("option: %s\n", arg)
        
        if arg == "list" {
            //fmt.Printf("list\n");
            reqSock := zmq.NewSock( zmq.Req )
            defer reqSock.Destroy()
            
            err := reqSock.Connect("tcp://127.0.0.1:7293")
            if err != nil {
               log.Panicf("error binding: %s", err)
               os.Exit(1)
            }
                        
            reqMsg := []byte("request")
            reqSock.SendMessage([][]byte{reqMsg})
            
            reply, err := reqSock.RecvMessage()
            if err != nil {
               log.Panicf("error receiving: %s", err)
               os.Exit(1)
            }
            
            fmt.Printf("reply: %s\n", string( reply[0] ) )
        }
        if arg == "pull" {
            pullSock := zmq.NewSock( zmq.Sub )
            defer pullSock.Destroy()
            
            err := pullSock.Connect("tcp://127.0.0.1:7294")
            if err != nil {
               log.Panicf("error binding: %s", err)
               os.Exit(1)
            }
            
            for {
                jsonMsg, err := pullSock.RecvMessage()
                if err != nil {
                   log.Panicf("error receiving: %s", err)
                   os.Exit(1)
                }
                
                fmt.Printf("pulled: %s\n", string( jsonMsg[0] ) )
                
                //var msg DevEvent
                //json.Unmarshal( jsonMsg[0], &msg )
            }
        }
        if arg == "server" {
            zmqReqRep()
            zmqPub( pubEventCh )
            var num int = 1
            for {
                devEvent := PubEvent{}
                devEvent.action = 0 // connect
                devEvent.name = "test"
                uuid := fmt.Sprintf("fakeuuid %d", num)
                devEvent.uuid = uuid
                num++
                pubEventCh <- devEvent
                
                time.Sleep( time.Second * 5 )
            }
        }
        
        return
    }
    
    zmqReqRep()
    zmqPub( pubEventCh )
    
    // Read in config
    configFile := "config.json"
    configFh, err := os.Open(configFile)   
      if err != nil {
          log.Panicf("failed reading file: %s", err)
      }
      defer configFh.Close()
      
    jsonBytes, _ := ioutil.ReadAll( configFh )
    var config Config
    json.Unmarshal( jsonBytes, &config )
    
    var vpnMissing bool = true
    tunName := getTunName()
    fmt.Printf("Tunnel name: %s\n", tunName)
    if tunName != "none" {
        vpnMissing = false
    }
    curIP := ifaceCurIP( tunName )
    fmt.Printf("Current IP on VPN: %s\n", curIP)
  
    cleanup_procs( config )
        
    devEventCh := make( chan DevEvent, 2 )
    runningDevs := make( map [string] RunningDev )
    baseProgs := BaseProgs{}
    
    // start web server waiting for trigger http command for device connect and disconnect
    
    var listen_addr = fmt.Sprintf( "%s:%d", config.CoordinatorHost, config.CoordinatorPort ) // "localhost:8027"
    go startServer( devEventCh, listen_addr )
    
    // start the 'osx_ios_device_trigger'
    go func() {
        fmt.Printf("Starting osx_ios_device_trigger\n");
        triggerCmd := exec.Command( config.DeviceTrigger )
        
        //triggerOut, _ := triggerCmd.StdoutPipe()
        //triggerCmd.Stdout = os.Stdout
        //triggerCmd.Stderr = os.Stderr
        err := triggerCmd.Start()
        if err != nil {
            fmt.Println(err.Error())
        } else {
            baseProgs.trigger = triggerCmd.Process
        }
        /*for {
          line, err := ioutil.Read(triggerOut)
          if err != nil {
            break
          }
        }*/
        triggerCmd.Wait()
        fmt.Printf("Ended: osx_ios_device_trigger\n");
    }()
	
    // start the video enabler
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
	
    if vpnMissing {
        fmt.Printf("VPN not enabled; skipping start of STF\n")
        baseProgs.stf = nil
    } else {
        // start stf and restart it when needed
        // TODO: if it doesn't restart / crashes again; give up
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
	
    SetupCloseHandler( runningDevs, &baseProgs, config )
    
    /*go func() {
      // repeatedly check vpn connection
          
      // when vpn goes down
        // log an error
        // wait for it to come back up
        // restart the various things to use the new IP
    }*/

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
                // start mirrorfeed
                mirrorPort := config.MirrorFeedPort // 8000
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
                
                //pipeHandle, _ := os.OpenFile( pipeName, os.O_RDWR, 0600 )
                
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
            
                // Sleep to ensure that video enabling process is finished before we try to start wdaproxy
                // This is needed because the USB device drops out and reappears during video enabling
                time.Sleep( time.Second * 9 )
            }
            
            // start wdaproxy
            wdaPort := config.WDAProxyPort // "8100"
            fmt.Printf("Starting wdaproxy\n")
            
            fmt.Printf("  wdaproxy -p %s -d -W %s -u %s\n", wdaPort, config.WDARoot, uuid )
            proxyCmd := exec.Command( "../../bin/wdaproxy", "-p", wdaPort, "-d", "-W", ".", "-u", uuid )
            //proxyCmd.Stdout = os.Stdout
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
                
                //proxyCmd.Wait()
                //wdaReader := bufio.NewReader( proxyPipe )
                wdaScanner := bufio.NewScanner( proxyPipe )
                //var line string
                for wdaScanner.Scan() {
                    //line, err := wdaReader.ReadString('\n')
                    //line, err := ioutil.Read(proxyPipe)
                    line := wdaScanner.Text()
                    
                    /*if err != nil {
                        break
                    }*/
                    if strings.Contains( line, "is implemented in both" ) {
                    } else if strings.Contains( line, "Couldn't write value" ) {
                    } else {
                        fmt.Println( line )
                    }
                }
                
                stopChannel<- true
                
                fmt.Printf("wdaproxy ended\n")
            }()
            
            time.Sleep( time.Second * 3 )
            
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

func cleanup_procs(config Config) {
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
    
    // The following does not work when DNS is not enabled with Tunnelblick :(
    /*cmd := "echo show State:/Network/OpenVPN | scutil | grep TunnelDevice | awk '{print $3}' | tr -d \"\\n\""
    out, err := exec.Command( "bash", "-c", cmd ).Output()
    if err != nil {
        return "none"
    }
    ret := string( out )
    if ret == "" {
        ret = "none"
    }*/
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

func zmqPub( pubEventCh <-chan PubEvent ) {
    var sentDummy bool = false
    
    // start the zmq pub mechanism
    go func() {
        pubSock := zmq.NewSock(zmq.Pub)
        //pubSock, _ := zmq.NewPub("tcp://127.0.0.1:7294/x")
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
            
            /*err = pubSock.SendFrame([]byte(reqMsg), zmq.FlagNone )
            if err != nil {
               log.Panicf("error encoding JSON: %s", err)
            }*/
            pubSock.SendMessage( [][]byte{ []byte("devEvent"), reqMsg} )
        }
    }()
}

func zmqReqRep() {
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
            /*msg, err := repSock.RecvMessage()
            if err != nil {
               log.Panicf("error receiving: %s", err)
               os.Exit(1)
            }
            
            fmt.Printf("Received: %s\n", string( msg[0] ) )
            
            response := []byte("response")
            repSock.SendMessage([][]byte{response})*/
            
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

func SetupCloseHandler( runningDevs map [string] RunningDev, baseProgs *BaseProgs, config Config ) {
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
        cleanup_procs( config)
        fmt.Println("Shutdown ok\n")
        
        os.Exit(0)
    }()
}

func getDeviceName( uuid string ) (string) {
    name, _ := exec.Command( "idevicename", "-u", uuid ).Output()
    if name == nil || len(name) == 0 {
        fmt.Printf("idevicename returned nothing for uuid %s\n", uuid)
    }
    nameStr := string(name)
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
    //fmt.Printf("len:%d\n", len(uuid) )
    if len(uuid) == 24 {
        p1 := uuid[:8]
        p2 := uuid[8:]
        uuid = fmt.Sprintf("%s-%s",p1,p2)
        //fmt.Printf("fixed:%s\n", uuid )
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