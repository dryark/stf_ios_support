package main

import (
    "bufio"
    "bytes"
    "context"
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
    "text/template"
    
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
    LogFile         string `json:"log_file"`
    LinesLogFile    string `json:"lines_log_file"`
    VpnName         string `json:"vpn_name"`
    NetworkInterface string `json:"network_interface"`
}

type VpnInfo struct {
    Err     string `json:"err"`
    IpAddr  string `json:"ipAddr"`
    TunName string `json:"tunName"`
}

type DevEvent struct {
    action int
    uuid   string
}

type PubEvent struct {
    action  int
    uuid    string
    name    string
    wdaPort int
    vidPort int
}

type RunningDev struct {
    uuid         string
    name         string
    mirror       *os.Process
    ff           *os.Process
    proxy        *os.Process
    device       *os.Process
    shuttingDown bool
    lock         sync.Mutex
    failed       bool
    wdaPort      int
    vidPort      int
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

type Vpn struct {
    state string
    autoConnect string
}

var gStop bool

func read_config( configPath string ) *Config {
    fh, serr := os.Stat( configPath )
    if serr != nil {
        log.WithFields( log.Fields{
            "type": "err_read_config",
            "error": serr,
            "config_path": configPath,
        } ).Fatal("Could not read specified config path")
    }    
    var configFile string
    switch mode := fh.Mode(); {
        case mode.IsDir():
            configFile = fmt.Sprintf("%s/config.json", configPath)
        case mode.IsRegular():
            configFile = configPath
    }
    
    configFh, err := os.Open( configFile )   
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

func proc_stf_provider( baseProgs *BaseProgs, curIP string, config *Config, lineLog *log.Entry ) {
    plog := log.WithFields( log.Fields{ "proc": "stf_provider" } )
    lineLog = lineLog.WithFields( log.Fields{
        "proc": "stf_provider",
    } )
    
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
        
            cmd := exec.Command( "/usr/local/opt/node@8/bin/node", "--inspect=127.0.0.1:9230", "runmod.js", "provider",
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
                    //fmt.Printf( "[PROVIDER] %s\n", line )
                    lineLog.WithFields( log.Fields{ "line": line } ).Info("")
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

func proc_mirrorfeed( config *Config, tunName string, devd *RunningDev, lineLog *log.Entry ) {
    plog := log.WithFields( log.Fields{
        "proc": "mirrorfeed",
        "uuid": devd.uuid,
    } )
    lineLog = lineLog.WithFields( log.Fields{
        "proc": "mirrorfeed",
        "uuid": devd.uuid,
    } )
    
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
            //cmd.Stderr = os.Stderr
            errPipe, _ := cmd.StderrPipe()
            
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
            
            go func() {
                scanner := bufio.NewScanner( errPipe )
                for scanner.Scan() {
                    line := scanner.Text()
                    lineLog.WithFields( log.Fields{ "line": line, "iserr": true } ).Info("")
                }
            } ()
            scanner := bufio.NewScanner( outputPipe )
            for scanner.Scan() {
                line := scanner.Text()
                //fmt.Printf( "[VIDFEED-] %s\n", line )
                lineLog.WithFields( log.Fields{ "line": line } ).Info("")
            }
            
            devd.mirror = nil
            
            plog.WithFields( log.Fields{  "type": "proc_end" } ).Warn("Ended: mirrorfeed")
            
            devd.lock.Lock()
            exit := devd.shuttingDown
            devd.lock.Unlock()
            if exit { break }
            
            // sleep before restart to prevent rapid failing attempts
            time.Sleep( time.Second * 5 )
        }
    }()
}

func proc_ffmpeg( config *Config, devd *RunningDev, devName string, lineLog *log.Entry ) {
    plog := log.WithFields( log.Fields{
        "proc": "ffmpeg",
        "uuid": devd.uuid,
    } )
    lineLog = lineLog.WithFields( log.Fields{
        "proc": "ffmpeg",
        "uuid": devd.uuid,
    } )
     
    if devd.shuttingDown {
        return
    }
    go func() {
        for {
            ops := []string{
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
                "pipe:1",
            }
            
            plog.WithFields( log.Fields{
                "type": "proc_start",
                "ops": ops,
            } ).Info("Starting: ffmpeg")
            
            cmd := exec.Command( "./run-ffmpeg.sh", ops... )
            
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
                    //fmt.Printf( "[FFMPEG--] %s\n", line )
                    lineLog.WithFields( log.Fields{ "line": line } ).Info("")
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

func proc_wdaproxy( 
        config *Config, 
        devd *RunningDev, 
        devEvent *DevEvent, 
        uuid string, 
        devName string, 
        pubEventCh chan<- PubEvent, 
        lineLog *log.Entry,
        iosVersion string ) {
    plog := log.WithFields( log.Fields{
        "proc": "wdaproxy",
        "uuid": devd.uuid,
    } )
    lineLog = lineLog.WithFields( log.Fields{
        "proc": "wdaproxy",
        "uuid": devd.uuid,
    } )

    // start wdaproxy
    wdaPort := config.WDAProxyPort
    
    if devd.shuttingDown {
        return
    }
    go func() {
        for {
            iversion := fmt.Sprintf("--iosversion=%s", iosVersion)
            ops := []string{
              "-p", strconv.Itoa( wdaPort ),
              "-q", strconv.Itoa( wdaPort ),
              "-d",
              "-W", ".",
              "-u", uuid,
              iversion,
            }
            
            plog.WithFields( log.Fields{
              "type": "proc_start",
              "port": wdaPort,
              "wda_folder": config.WDARoot,
              "ops": ops,
            } ).Info("Starting wdaproxy")
            
            cmd := exec.Command( "../../bin/wdaproxy", ops... )
            
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
                    } else {
                        lineLog.WithFields( log.Fields{ "line": line } ).Info("")
                        //fmt.Printf( "[WDAPROXY] %s\n", line )
                    }
                }
            } ()
            scanner := bufio.NewScanner( errPipe )
            for scanner.Scan() {
                line := scanner.Text()
                
                if strings.Contains( line, "[WDA] successfully started" ) {
                    plog.WithFields( log.Fields{ "type": "wda_started" } ).Info("WDA started")
                    //fmt.Printf( "[WDAPROXE] %s\n", line )
                }
                
                lineLog.WithFields( log.Fields{ "line": line, "iserr": true } ).Info("")
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

func proc_device_ios_unit( config *Config, devd *RunningDev, uuid string, curIP string, lineLog *log.Entry ) {
    plog := log.WithFields( log.Fields{
      "proc": "stf_device_ios",
      "uuid": uuid,
    } )
    lineLog = lineLog.WithFields( log.Fields{
        "proc": "stf_device_ios",
        "uuid": devd.uuid,
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
            
            cmd := exec.Command( "/usr/local/opt/node@8/bin/node",
                "--inspect=0.0.0.0:9240",
                "runmod.js", "device-ios",
                "--serial", uuid,
                "--connect-push", pushStr,
                "--connect-sub", subStr,
                "--public-ip", curIP,
                "--wda-port", strconv.Itoa( devd.wdaPort ),
                "--screen-ws-url-pattern", fmt.Sprintf( "wss://%s/frames/%s/%d/x", config.STFHostname, curIP, devd.vidPort ),
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
                //fmt.Printf( "[StfDvIos] %s\n", line )
                if strings.Contains( line, "Now owned by" ) {
                    pos := strings.Index( line, "Now owned by" )
                    pos += len( "Now owned by" ) + 2
                    ownedStr := line[ pos: ]
                    endpos := strings.Index( ownedStr, "\"" )
                    owner := ownedStr[ :endpos ]
                    plog.WithFields( log.Fields{
                        "type": "wda_owner_start",
                        "owner": owner,
                    } ).Info("Device Owner Start")
                }
                if strings.Contains( line, "No longer owned by" ) {
                    pos := strings.Index( line, "No longer owned by" )
                    pos += len( "No longer owned by" ) + 2
                    ownedStr := line[ pos: ]
                    endpos := strings.Index( ownedStr, "\"" )
                    owner := ownedStr[ :endpos ]
                    plog.WithFields( log.Fields{
                        "type": "wda_owner_stop",
                        "owner": owner,
                    } ).Info("Device Owner Stop")
                }
                        
                lineLog.WithFields( log.Fields{ "line": line } ).Info("")
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

type HupData struct {
    hupA bool
    hupB bool
    mutex sync.Mutex
}

type JSONLog struct {
	  file      *os.File
	  fileName  string
	  formatter *log.JSONFormatter
	  failed    bool
	  hupData   *HupData
	  id        int
}
func ( hook *JSONLog ) Fire( entry *log.Entry ) error {
    // If we have failed to write to the file; don't bother trying
    if hook.failed { return nil }
    
    jsonformat, _ := hook.formatter.Format( entry )
    
    fh := hook.file
    
    doHup := false
    hupData := hook.hupData
    hupData.mutex.Lock()
    if hook.id == 1 {
        doHup = hupData.hupA
        if doHup { hupData.hupA = false }
    } else if hook.id == 2 {
        doHup = hupData.hupB
        if doHup { hupData.hupB = false }
    }
    hupData.mutex.Unlock()
    
    if doHup {
        fh.Close()
        fhnew, err := os.OpenFile( hook.fileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666 )
        if err != nil {
            fmt.Fprintf( os.Stderr, "Unable to open file for writing: %v", err )
            fh = nil
        }
        fh = fhnew
        hook.file = fh
        /*log.WithFields( log.Fields{
            "type": "sighup",
            "state": "reopen",
            "file": hook.fileName,
        } ).Info("HUP requested")*/
        fmt.Fprintf( os.Stdout, "Hup %s\n", hook.fileName )
    }
    
    var err error
    if entry.Context != nil {
        // There is context; this is meant for the lines logfile
        str := string( jsonformat )
        str = strings.Replace( str, "\"level\":\"info\",", "", 1 )
        str = strings.Replace( str, "\"msg\":\"\",", "", 1 )
        _, err = fh.WriteString( str )
    } else {
        _, err = fh.WriteString( string( jsonformat ) )
    }
    
    if err != nil {
        hook.failed = true
        fmt.Fprintf( os.Stderr, "Cannot write to logfile: %v", err )
        return err
    }
  
    return nil
}
func (hook *JSONLog) Levels() []log.Level {
    return []log.Level{ log.PanicLevel, log.FatalLevel, log.ErrorLevel, log.WarnLevel, log.InfoLevel, log.DebugLevel }
}
func AddJSONLog( logger *log.Logger, fileName string, id int, hupData *HupData ) {
    logFile, err := os.OpenFile( fileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666 )
    if err != nil {
        fmt.Fprintf( os.Stderr, "Unable to open file for writing: %v", err )
    }
    
    fileHook := JSONLog{
        file: logFile,
        fileName: fileName,
        formatter: &log.JSONFormatter{},
        failed: false,
        hupData: hupData,
        id: id,
    }
    
    if logger == nil {
        log.AddHook( &fileHook )
    } else {
        logger.AddHook( &fileHook )
    }
}

type DummyWriter struct {
}
func (self *DummyWriter) Write( p[]byte) (n int, err error) {
    return len(p), nil
}

func setup_log( config *Config, debug bool, jsonLog bool ) (*log.Entry) {
    if jsonLog {
        log.SetFormatter( &log.JSONFormatter{} )
    }
    
    lineLogger1 := log.New()
    //lineLogger1.SetFormatter( nil )
    dummyWriter := DummyWriter{}
    lineLogger1.SetOutput( &dummyWriter )
    lineLogger := lineLogger1.WithContext( context.Background() )
    
    if debug {
        log.WithFields( log.Fields{ "type": "debug_status" } ).Warn("Debugging enabled")
        log.SetLevel( log.DebugLevel )
        lineLogger1.SetLevel( log.DebugLevel )
    } else {
        log.SetLevel( log.InfoLevel )
        lineLogger1.SetLevel( log.InfoLevel )
    }
    
    hupData := coro_sighup()
    
    AddJSONLog( nil, config.LogFile, 1, hupData )
    AddJSONLog( lineLogger1, config.LinesLogFile, 2, hupData )
    
    return lineLogger
}

func run_osa( app string, cmds ...string ) (string){
    apptell := fmt.Sprintf(`tell application "%s"`, app )
    args := []string{ "-e", apptell }
    for _, cmd := range cmds {
        args = append( args, "-e" )
        args = append( args, cmd )
    }
    args = append( args, "-e", "end tell" )
    
    res, err := exec.Command("/usr/bin/osascript", args...).Output()
    if err != nil {
        fmt.Printf( err.Error() )
        return ""
    }
    
    // Remove the ending carriage return and then return
    resStr := string(res)
    return resStr[:len(resStr)-1]
}

func tblick_osa( cmds ...string ) (string) {
    return run_osa( "Tunnelblick", cmds... )
}

func vpn_names() ( []string ) {
    vpnsStr := tblick_osa( "get configurations" )
    vpns := strings.Split( vpnsStr, ", " )
    var justVpns []string
    for _, vpn := range vpns {
        parts := strings.Split(vpn," ")
        justVpns = append( justVpns, parts[1] )
    }
    return justVpns
}

func vpn_state( confName string ) (string) {
    req := fmt.Sprintf(`get state of first configuration where name="%s"`, confName)
    state := tblick_osa( req )
    return state
}

func vpn_connect( confName string ) {
    req := fmt.Sprintf(`connect "%s"`, confName)
    tblick_osa( req )
    for i := 0; i <= 10; i++ {
        time.Sleep( time.Millisecond * 200 )
        if vpn_state( confName ) == "CONNECTED" {
            return
        }
    }
}

func vpn_states() ([]string) {
    stateStr := tblick_osa( "get state of configurations" )
    states := strings.Split( stateStr, ", " )
    
    var stateArr []string
    for _, state := range states {
        stateArr = append( stateArr, state )
    }
    return stateArr
}

func vpn_autoconnects() ([]string) {
    stateStr := tblick_osa( "get autoconnect of configurations" )
    states := strings.Split( stateStr, ", " )
    
    var stateArr []string
    for _, state := range states {
        stateArr = append( stateArr, state )
    }
    return stateArr
}

func vpns_getall() ( map [string] *Vpn ) {
    vpns := make( map [string] *Vpn )
    
    names        := vpn_names()
    states       := vpn_states()
    autoconnects := vpn_autoconnects()
    
    for i, name := range names {
        state := states[i]
        if state == "EXITING" { state = "DISCONNECTED" }
        auto := autoconnects[i]
        aVpn := Vpn{
          state: state,
          autoConnect: auto,
        }
        vpns[ name ] = &aVpn
    }
    
    return vpns
}

func check_vpn_status( config *Config ) {
    vpnName := config.VpnName
    
    if _, err := os.Stat("/Applications/Tunnelblick.app"); os.IsNotExist(err) {
        // Tunnelblick is not installed; don't try to call it
        log.WithFields( log.Fields{ "type": "vpn_error" } ).Error("Tunnelblick is not installed")
        return
    }
    
    vpns := vpns_getall()
    if vpn, exists := vpns[ vpnName ]; exists {
        if vpn.state == "DISCONNECTED" {
            warning := fmt.Sprintf("Specified VPN \"%s\" not connected; connecting", vpnName)
            log.WithFields( log.Fields{ "type": "vpn_warn", "vpnName": vpnName } ).
                Warn( warning )
            vpn_connect( vpnName )
        }
    } else {
        error := fmt.Sprintf( "Specified VPN \"%s\" is not setup in Tunnelblick", vpnName )
        log.WithFields( log.Fields{ "type": "vpn_error", "vpnName": vpnName } ).
            Error( error )
    }
}

func main() {
    gStop = false
    
    var debug = flag.Bool( "debug", false, "Use debug log level" )
    var jsonLog = flag.Bool( "json", false, "Use json log output" )
    var vpnlist = flag.Bool( "vpnlist", false, "List VPNs then exit" )
    var configFile = flag.String( "config", "config.json", "Config file path" )
    flag.Parse()
    
    if *vpnlist {
        vpns := vpns_getall()
        
        for vpnName, vpn := range vpns {
            fmt.Printf("Name: %s - Autoconnect: %s - %s\n", vpnName, vpn.autoConnect, vpn.state)
        }
        os.Exit(0)
    }
    
    config := read_config( *configFile )
    
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
    
    var ifName string
    var curIP string
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
    wdaPorts := construct_ports( config, config.WDAPorts )
    vidPorts := construct_ports( config, config.VidPorts )
    baseProgs := BaseProgs{}
    baseProgs.shuttingDown = false
    
    coro_http_server( config, devEventCh, &baseProgs, runningDevs )
    proc_device_trigger( config, &baseProgs )
    if !config.SkipVideo {
        ensure_proper_pipe( config )
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
    event_loop( config, curIP, devEventCh, ifName, pubEventCh, runningDevs, wdaPorts, vidPorts, lineLog )
}

func ensure_proper_pipe( config *Config ) {
    file := config.Pipe
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
    if mode != os.ModeNamedPipe {
        log.WithFields( log.Fields{
            "type": "pipe_fix",
            "pipe_file": file,
        } ).Info("Pipe was incorrect type; deleting and recreating as fifo")
        // delete the file then create it properly as a pipe
        os.Remove( file )
        syscall.Mkfifo( file, 0600 )
    }
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

func coro_http_server( config *Config, devEventCh chan<- DevEvent, baseProgs *BaseProgs, runningDevs map [string] *RunningDev ) {
    // start web server waiting for trigger http command for device connect and disconnect
    var listen_addr = fmt.Sprintf( "0.0.0.0:%d", config.CoordinatorPort )
    go startServer( devEventCh, listen_addr, baseProgs, runningDevs )
}

func ifAddr( ifName string ) ( addrOut string ) {
    ifaces, err := net.Interfaces()
    if err != nil {
        fmt.Printf( err.Error() )
        os.Exit( 1 )
    }
    
    addrOut = ""
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
            if iface.Name == ifName {
                addrOut = ip.String()
            }
        }
    }
    return addrOut
}

func vpn_info( config *Config ) ( string, string, string ) {
    jsonBytes, _ := exec.Command( "./tblick-info.sh", config.VpnName ).Output()
    vpnInfo := VpnInfo{}
    err := json.Unmarshal( jsonBytes, &vpnInfo )
    if err != nil {
        fmt.Printf( err.Error() )
    }
    return vpnInfo.TunName, vpnInfo.IpAddr, vpnInfo.Err
}

func get_net_info( config *Config ) ( string, string, bool ) {
    var vpnMissing bool = false
    
    // This information comes from Tunnelblick log
    // It may no longer be active
    tunName, curIP, err := vpn_info( config )
    
    if err != "" {
        log.WithFields( log.Fields{
            "type": "vpn_err",
            "err": err,
        } ).Info( err )
        return "", "", true
    }
    
    log.WithFields( log.Fields{
        "type": "info_vpn",
        "tunnel_name": tunName,
    } ).Info("Tunnel name")
    
    ipConfirm := getTunIP( tunName )
    if ipConfirm != curIP {
        // The tunnel is no longer active
        vpnMissing = true
    } else {
        log.WithFields( log.Fields{
            "type": "info_vpn",
            "tunnel_name": tunName,
            "ip": curIP,
        } ).Info("IP on VPN")
    }
    
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
        vidPorts map [int] *PortItem,
        lineLog *log.Entry ) {
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
                // TODO log an error
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
        if cmd[0] == "/usr/local/opt/node@8/bin/node" && cmd[3] == "device-ios" {
            plog.WithFields( log.Fields{
                "proc": "device-ios",
            } ).Debug("Leftover Proc - Sending SIGTERM")
            
            syscall.Kill( proc.Pid, syscall.SIGTERM )
        }
        
        // node --inspect=[ip]:[port] runmod.js provider
        if cmd[0] == "/usr/local/opt/node@8/bin/node" && cmd[3] == "provider" {
            plog.WithFields( log.Fields{
                "proc": "stf_provider",
            } ).Debug("Leftover Proc - Sending SIGTERM")
            
            syscall.Kill( proc.Pid, syscall.SIGTERM )
        }
    }
}

func getTunIP( iface string ) string {
    cmd := fmt.Sprintf( "echo show State:/Network/Interface/%s/IPv4 | scutil | grep ' Addresses' -A 1 | tail -1 | cut -d \\  -f 7 | tr -d \"\\n\"", iface )
    out, _ := exec.Command( "bash", "-c", cmd ).Output()
    return string( out )
}

func ifaceCurIP( tunName string ) string {
    ipStr := ifAddr( tunName )
    if ipStr != "" {
        log.WithFields( log.Fields{
            "type": "net_interface_info",
            "interface_name": tunName,
            "ip": ipStr,
        } ).Debug("Interface Details")
    } else {
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

func coro_sigterm( runningDevs map [string] *RunningDev, baseProgs *BaseProgs, config *Config ) {
    c := make(chan os.Signal, 2)
    signal.Notify(c, os.Interrupt, syscall.SIGTERM)
    go func() {
        <- c
        log.WithFields( log.Fields{
            "type": "sigterm",
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
            "type": "sigterm",
            "state": "done",
        } ).Info("Shutdown finished")
        
        os.Exit(0)
    }()
}

func coro_sighup() ( *HupData ) {
    hupData := HupData{
        hupA: false,
        hupB: false,
    }
    c := make(chan os.Signal, 2)
    signal.Notify(c, os.Interrupt, syscall.SIGHUP)
    go func() {
        for {
            <- c
            log.WithFields( log.Fields{
                "type": "sighup",
                "state": "begun",
            } ).Info("HUP requested")
            hupData.mutex.Lock()
            hupData.hupA = true
            hupData.hupB = true
            hupData.mutex.Unlock()
        }
    }()
    return &hupData
}

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
        if i > 30 {
            log.WithFields( log.Fields{
                "type": "ilib_getinfo_fail",
                "uuid": uuid,
                "key": keyName,
                "try": i,
            } ).Debug("ideviceinfo failed after 30 attempts over 10 seconds")
            return "" 
        }
        
        ops := []string{
          "-u", uuid,
        }
        if keyName != "" {
          ops = append( ops, "-k", keyName )
        }
        
        name, _ := exec.Command( "/usr/local/bin/ideviceinfo", ops... ).Output()
        if name == nil || len(name) == 0 {
            log.WithFields( log.Fields{
                "type": "ilib_getinfo_fail",
                "uuid": uuid,
                "key": keyName,
                "try": i,
            } ).Debug("ideviceinfo returned nothing")
    
            time.Sleep( time.Millisecond * 300 )
            continue
        }
        nameStr = string( name )
        break
    }
    nameStr = nameStr[:len(nameStr)-1]
    return nameStr
}

func startServer( devEventCh chan<- DevEvent, listen_addr string, baseProgs *BaseProgs, runningDevs map [string] *RunningDev ) {
    log.WithFields( log.Fields{
        "type": "http_start",
    } ).Info("HTTP started")
        
    rootClosure := func( w http.ResponseWriter, r *http.Request ) {
        handleRoot( w, r, baseProgs, runningDevs )
    }
    devinfoClosure := func( w http.ResponseWriter, r *http.Request ) {
        reqDevInfo( w, r, baseProgs, runningDevs )
    }
    http.HandleFunc( "/devinfo", devinfoClosure )
    http.HandleFunc( "/", rootClosure )
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

func reqDevInfo( w http.ResponseWriter, r *http.Request, baseProgs *BaseProgs, runningDevs map [string] *RunningDev ) {
    query := r.URL.Query()
    uuid := query.Get("uuid")
    info := getAllDeviceInfo( uuid )
    
    names := map[string]string{
        "DeviceName": "Device Name",
        "EthernetAddress": "MAC",
        "InternationalCircuitCardIdentity": "ICCI",
        "InternationalMobileEquipmentIdentity": "IMEI",
        "InternationalMobileSubscriberIdentity": "IMSI",
        "ModelNumber": "Model",
        //"HardwareModel": "Hardware Model",
        "PhoneNumber": "Phone Number",
        "ProductType": "Product",
        "ProductVersion": "IOS Version",
        "UniqueDeviceID": "Wifi MAC",        
    }
    
    for key, descr := range names {
        val := info[key]
        fmt.Fprintf( w, "%s: %s<br>\n", descr, val )
    }
}

func handleRoot( w http.ResponseWriter, r *http.Request, baseProgs *BaseProgs, runningDevs map [string] *RunningDev ) {
    device_trigger := "<font color='green'>on</font>"
    if baseProgs.trigger == nil { device_trigger = "off" }
    video_enabler := "<font color='green'>on</font>"
    if baseProgs.vidEnabler == nil { video_enabler = "off" }
    stf := "<font color='green'>on</font>"
    if baseProgs.stf == nil { stf = "off" }
    
    devOut := ""
    for _, dev := range runningDevs {
        mirror := "<font color='green'>on</font>"
        if dev.mirror == nil { mirror = "off" }
        
        ff := "<font color='green'>on</font>"
        if dev.ff == nil { ff = "off" }
        
        proxy := "<font color='green'>on</font>"
        if dev.proxy == nil { proxy = "off" }
        
        device := "<font color='green'>on</font>"
        if dev.device == nil { device = "off" }
        
        var str bytes.Buffer
        deviceTpl.Execute( &str, map[string]string{
            "uuid": "<a href='/devinfo?uuid=" + dev.uuid + "'>" + dev.uuid + "</a>",
            "name": dev.name,
            "mirror": mirror,
            "ff": ff,
            "proxy": proxy,
            "device": device,
        } )
        devOut += str.String()
    }
    
    rootTpl.Execute( w, map[string]string{
        "device_trigger": device_trigger,
        "video_enabler": video_enabler,
        "stf": stf,
        "devices": devOut,
    } )
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

var deviceTpl = template.Must(template.New("device").Parse(`
<table border=1 cellpadding=5 cellspacing=0>
  <tr>
    <td>UUID</td>
    <td>{{.uuid}}</td>
  </tr>
  <tr>
    <td>Name</td>
    <td>{{.name}}</td>
  </tr>
  <tr>
    <td>Video Mirror</td>
    <td>{{.mirror}}</td>
  </tr>
  <tr>
    <td>FFMpeg</td>
    <td>{{.ff}}</td>
  </tr>
  <tr>
    <td>WDA Proxy</td>
    <td>{{.proxy}}</td>
  </tr>
  <tr>
    <td>STF Device-IOS Unit</td>
    <td>{{.device}}</td>
  </tr>
</table>
`))

var rootTpl = template.Must(template.New("root").Parse(`
<!DOCTYPE html>
<html>
	<head>
	</head>
	<body>
	Base Processes:
	<table border=1 cellpadding=5 cellspacing=0>
	  <tr>
	    <td>Device Trigger</td>
	    <td>{{.device_trigger}}</td>
	  </tr>
	  <tr>
	    <td>Video Enabler</td>
	    <td>{{.video_enabler}}</td>
	  </tr>
	  <tr>
	    <td>STF</td>
	    <td>{{.stf}}</td>
	  </tr>
  </table><br><br>
	
	Devices:<br>{{.devices}}
	</body>
</html>
`))