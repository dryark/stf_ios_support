package main

import (
  "fmt"
  "net"
  "os"
  "os/exec"
  "strings"
  "regexp"
  log "github.com/sirupsen/logrus"
)

func getDefaultIf() ( string ) {
	out, err := exec.Command( "/usr/sbin/netstat", "-nr", "-f", "inet" ).Output()
    if err != nil {
        fmt.Printf("Error from netstat: %s\n", err )
        return ""
    }
    lines := strings.Split( string(out), "\n" )
    iFace := ""
    space := regexp.MustCompile(`\s+`)
    		
    for _, line := range lines {
    	if strings.Contains( line, "default " ) {
    		line = space.ReplaceAllString( line, " " )
    		
    		parts := strings.Split( line, " " )
    		if parts[0] == "default" {
    			iFace = parts[3]
    		}
    	}
    }
    return iFace
}

func ifAddr( ifName string ) ( addrOut string, okay bool ) {
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
                str := ip.String()
                if !strings.Contains(str,":") {
                    addrOut = str
                }
            }
        }
    }
    if addrOut != "" {
    	return addrOut, true
    }
    fmt.Printf("Network interface %s not found, exiting\n", ifName )
    return "", false
}

func get_net_info( config *Config ) ( string, string, bool ) {
    var vpnMissing bool = false

    // This information comes from Tunnelblick log
    // It may no longer be active
    tunName, curIP, err := vpn_info( config )

    if err != "" {
        log.WithFields( log.Fields{
            "type": "vpn_err",
            "err":  err,
        } ).Info( err )
        return "", "", true
    }

    log.WithFields( log.Fields{
        "type": "info_vpn",
        "interface_name": tunName,
    } ).Info("VPN Info - interface")

    ipConfirm := getTunIP( tunName )
    if ipConfirm != curIP {
        // The tunnel is no longer active
        vpnMissing = true
    } else {
        log.WithFields( log.Fields{
            "type":           "info_vpn",
            "interface_name": tunName,
            "ip":             curIP,
        } ).Info("VPN Info - ip")
    }

    return tunName, curIP, vpnMissing
}

func ifaceCurIP( tunName string ) string {
    ipStr, _ := ifAddr( tunName )
    if ipStr != "" {
        log.WithFields( log.Fields{
            "type":           "net_interface_info",
            "interface_name": tunName,
            "ip":             ipStr,
        } ).Debug("Interface Details")
    } else {
        log.WithFields( log.Fields{
            "type":           "err_net_interface",
            "interface_name": tunName,
        } ).Fatal("Could not find interface")
    }

    return ipStr
}