package main

import (
  "encoding/json"
  "fmt"
  "os"
  "os/exec"
  "strings"
  "time"
  log "github.com/sirupsen/logrus"
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

func vpn_info( config *Config ) ( string, string, string ) {
    jsonBytes, _ := exec.Command( "./tblick-info.sh", config.VpnName ).Output()
    vpnInfo := VpnInfo{}
    err := json.Unmarshal( jsonBytes, &vpnInfo )
    if err != nil {
        fmt.Printf( err.Error() )
    }
    return vpnInfo.TunName, vpnInfo.IpAddr, vpnInfo.Err
}

func getTunIP( iface string ) string {
    cmd := fmt.Sprintf( "echo show State:/Network/Interface/%s/IPv4 | scutil | grep ' Addresses' -A 1 | tail -1 | cut -d \\  -f 7 | tr -d \"\\n\"", iface )
    out, _ := exec.Command( "bash", "-c", cmd ).Output()
    return string( out )
}