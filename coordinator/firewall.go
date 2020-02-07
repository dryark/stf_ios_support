package main

import (
  "fmt"
  "os"
  "os/exec"
  "strings"
  log "github.com/sirupsen/logrus"
)

func firewall_ensureperm( findBin string ) {
    hasPerm := firewall_hasperm( findBin )
    if hasPerm {
        log.Warn("App already has firewall permissions: ", findBin )
        return
    }
    firewall_stop()
    firewall_addperm( findBin )
    firewall_start()
}

func firewall_hasperm( findBin string ) (bool) {
    curBins := firewall_getperms()
    for _, bin := range curBins {
        if bin == findBin {
            return true
        }
    }
    return false
}

func firewall_addperm( binary string ) {
    cmd := exec.Command( "/usr/libexec/ApplicationFirewall/socketfilterfw", "--add", binary )
    cmd.Stderr = os.Stderr
    cmd.Stdout = os.Stdout
    err := cmd.Run()
    if err != nil {
        log.Fatal( err )
    }
}

func firewall_stop() {
    cmd := exec.Command( "/usr/libexec/ApplicationFirewall/socketfilterfw", "--setglobalstate", "off" )
    cmd.Stderr = os.Stderr
    cmd.Stdout = os.Stdout
    err := cmd.Run()
    if err != nil {
        log.Fatal( err )
    }
}

func firewall_start() {
    cmd := exec.Command( "/usr/libexec/ApplicationFirewall/socketfilterfw", "--setglobalstate", "on" )
    cmd.Stderr = os.Stderr
    cmd.Stdout = os.Stdout
    err := cmd.Run()
    if err != nil {
        log.Fatal( err )
    }
}

func firewall_delperm( binary string ) {
    cmd := exec.Command( "/usr/libexec/ApplicationFirewall/socketfilterfw", "--remove", binary )
    cmd.Stderr = os.Stderr
    cmd.Stdout = os.Stdout
    err := cmd.Run()
    if err != nil {
        log.Fatal( err )
    }
}

func firewall_showperms() {
    bytes, _ := exec.Command( "/usr/libexec/ApplicationFirewall/socketfilterfw", "--listapps" ).Output()
    fmt.Printf( string( bytes ) )
}

func firewall_getperms() ( [] string ) {
    bytes, _ := exec.Command( "/usr/libexec/ApplicationFirewall/socketfilterfw", "--listapps" ).Output()
    
    lines := strings.Split( string(bytes), "\n" )

    var apps []string
    app := ""
    for _, line := range lines {
        colonPos := strings.Index( line, ":" )
        if colonPos != -1 {
            app = line[ colonPos + 3 : len( line ) - 1 ]
        } else {
            allowPos := strings.Index( line, "( Allow" )
            if allowPos != -1 {
                apps = append( apps, app )
            }
        }
    }
    log.Debug( "Cureent apps with permissions", apps )
    return apps
}