package main

import (
  "bufio"
  "bytes"
  "encoding/json"
  "fmt"
  "os"
  "os/exec"
  //"path/filepath"
  "strings"
  "time"
  log "github.com/sirupsen/logrus"
  fsnotify "github.com/fsnotify/fsnotify"
)

type VpnInfo struct {
    Err     string `json:"err"`
    IpAddr  string `json:"ipAddr"`
    TunName string `json:"tunName"`
}

type Vpn struct {
    state string
    autoConnect string
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

type VpnEvent struct {
    action int
    text1  string
    text2  string
}

func openvpn_NewLauncher( config *Config ) (*Launcher) {
    vpnConfig := config.Vpn.OvpnConfig
  
    arguments := []string {
        config.BinPaths.Openvpn,
        "--config", vpnConfig,
    }
    
    label := fmt.Sprintf("com.tmobile.coordinator.openvpn")
    wd := config.Vpn.OvpnWd
    keepalive := true
    asRoot := true
    vpnLauncher := NewLauncher( label, arguments, keepalive, wd, asRoot )
    //vpnLauncher.stdout = config.WDAWrapperStdout
    //vpnLauncher.stderr = config.WDAWrapperStderr
    
    return vpnLauncher
}

func openvpn_load( config *Config ) {
    vpnLauncher := openvpn_NewLauncher( config )
    vpnLauncher.load()
}

func openvpn_unload( config *Config ) {
    vpnLauncher := openvpn_NewLauncher( config )
    vpnLauncher.unload()
}

func check_vpn_status( config *Config, baseProgs *BaseProgs, vpnEventCh chan<- VpnEvent ) {
    vpnType := config.Vpn.VpnType
    if vpnType == "tunnelblick" {
        vpnName := config.Vpn.TblickName
    
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
    } else if vpnType == "openvpn" {
        vpnLauncher := openvpn_NewLauncher( config )
        
        //wd, _ := os.Getwd()
        //abs, _ := filepath.Abs( "logs/openvpn.log" )
        //log.Info("current directory: ", wd, ", absolute path: ", abs )
        
        // TODO: create file if it doesn't exist
        vpnLog := "logs/openvpn.log"
        fh, _ := os.Open( vpnLog )
        scanForLastInterface( bufio.NewScanner( fh ), vpnEventCh )   
        
        curPos, _ := fh.Seek( 0, os.SEEK_END )
        
        vpnPid := vpnLauncher.pid()
        if vpnPid == 0 {
            log.WithFields( log.Fields{ "type": "openvpn_not_loaded" } ).Fatal("OpenVPN is not running")
        }
        
        watcher, err := fsnotify.NewWatcher()
        if err != nil {
            log.Fatal(err)
        }
        baseProgs.vpnLogWatcher = watcher
        defer watcher.Close()
        
        doneChan := make( chan bool )
        baseProgs.vpnLogWatcherStopChan = doneChan
        go func() {
            for {
                select {
                    case event, ok := <- watcher.Events:
                        if !ok {
                            baseProgs.vpnLogWatcherStopChan = nil
                            return
                        }
                        if event.Op & fsnotify.Write == fsnotify.Write {
                            //modFile = event.Name
                            newPos, _ := fh.Seek( 0, os.SEEK_CUR )
                            dif := newPos - curPos
                            if dif > 0 {
                                log.WithFields( log.Fields{ "type": "vpn_log_data", "size": dif } ).Info("New vpn log data")
                                fh.Seek( curPos, os.SEEK_SET )
                                buf := make( []byte, dif )
                                /*n, _ := */fh.Read( buf )
                                reader := bytes.NewReader( buf )
                                scanForInterface( bufio.NewScanner( reader ), vpnEventCh )            
                            }
                        }
                    case err, ok := <- watcher.Errors:
                        if !ok && err != nil {
                            log.Error(err)
                            baseProgs.vpnLogWatcherStopChan = nil
                            return
                        }
                    case res := <- doneChan:
                        if res == true {
                            baseProgs.vpnLogWatcherStopChan = nil
                            return
                        }
                }
            }
        }()
        
        err = watcher.Add( vpnLog )
        if err != nil {
            log.Fatal(err)
        }
    }
}

func scanForInterface( scanner *bufio.Scanner, vpnEventCh chan<- VpnEvent ) {
    for scanner.Scan() {
        line := scanner.Text()
        //log.WithFields( log.Fields{ "type": "vpn_log_line", "line": line } ).Info("VPN log line")
        if strings.Contains( line, "ifconfig" ) && strings.HasSuffix( line, " up" ) {
            interfaceNotice( uplineToInterface( line ), vpnEventCh )
        }
    }
}

func scanForLastInterface( scanner *bufio.Scanner, vpnEventCh chan<- VpnEvent ) {
    lastUpLine := ""
    for scanner.Scan() {
        line := scanner.Text()
        //log.WithFields( log.Fields{ "type": "vpn_log_line", "line": line } ).Info("VPN log line")
        if strings.Contains( line, "ifconfig" ) && strings.HasSuffix( line, " up" ) {
            lastUpLine = line 
        }
    }
    if lastUpLine != "" {
        interfaceNotice( uplineToInterface( lastUpLine ), vpnEventCh )
    }
}

func interfaceNotice( iface string, vpnEventCh chan<- VpnEvent ) {
    log.WithFields( log.Fields{ "type": "vpn_iface", "iface": iface } ).Info("VPN interface")
    vpnEvent := VpnEvent{
        action: 0,
        text1: iface,
    }
    vpnEventCh <- vpnEvent
}

func uplineToInterface( line string ) ( string ) {
    // ... /sbin/ifconfig utun1 ... up
    ifpos := strings.Index( line, "ifconfig" )
    ifpos += 9
    iface := line[ ifpos: ]
    ifend := strings.Index( iface, " " )
    iface = iface[ 0: ifend ]
    return iface
}

func vpn_shutdown( baseProgs *BaseProgs ) {
    if baseProgs.vpnLogWatcher != nil {
        baseProgs.vpnLogWatcher.Close()
        if baseProgs.vpnLogWatcherStopChan != nil {
            baseProgs.vpnLogWatcherStopChan <- true
        }
    }
}

func vpn_info( config *Config ) ( string, string, string ) {
    vpnType := config.Vpn.VpnType
    if vpnType == "tunnelblick" {
        jsonBytes, _ := exec.Command( "./tblick-info.sh", config.Vpn.TblickName ).Output()
        vpnInfo := VpnInfo{}
        err := json.Unmarshal( jsonBytes, &vpnInfo )
        if err != nil {
            fmt.Printf( err.Error() )
        }
        return vpnInfo.TunName, vpnInfo.IpAddr, vpnInfo.Err
    } else if vpnType == "openvpn" {
        return "", "", ""
    }
    return "", "", ""
}

func getTunIP( iface string ) string {
    cmd := fmt.Sprintf( "echo show State:/Network/Interface/%s/IPv4 | scutil | grep ' Addresses' -A 1 | tail -1 | cut -d \\  -f 7 | tr -d \"\\n\"", iface )
    out, _ := exec.Command( "bash", "-c", cmd ).Output()
    return string( out )
}