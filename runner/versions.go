package main

import (
    uj "github.com/nanoscopic/ujsonin/mod"
    "io/ioutil"
    "os/exec"
    "strings"
    "os"
    "fmt"
)

func loadVersionInfo( vmap map[string] VersionInfo ) {
    if !fileExists( "bin/bins.json" ) {
        fmt.Printf("bin/bins.json file does not exist; cannot do extended version check")
        return
    }
    
	content, _ := ioutil.ReadFile("bin/bins.json")
    root, _ := uj.Parse( content )
    
    bins := root.Get("bins")
    
    bins.ForEach( func( item *uj.JNode ) {
		//short := item.Get("short").String()
		name := item.Get("name").String()
		cmd := item.Get("cmd").String()
		
		arr := strings.Split( "./bin/" + cmd, " " )
		out, err := exec.Command(arr[0], arr[1:]...).Output()

        if err != nil {
            fmt.Printf("err running %s : %s\n", "./bin/" + cmd, err )
            return
        }
        
        vmap[ name ] = processVersionText( string(out) ) 
    } )
}

func processVersionText( text string ) ( VersionInfo ) {
    lines := strings.Split( text, "\n" )
    var res VersionInfo = VersionInfo{}
    for _,line := range lines {
        if strings.HasPrefix( line, "Commit:"  ) { res.GitCommit   = line[7:] }
        if strings.HasPrefix( line, "Date:"    ) { res.GitDate     = line[5:] }
        if strings.HasPrefix( line, "Remote:"  ) { res.GitRemote   = line[7:] }
        if strings.HasPrefix( line, "Version:" ) { res.EasyVersion = line[8:] }
    }
    return res
}

func fileExists(filename string) bool {
    info, err := os.Stat(filename)
    if os.IsNotExist(err) {
        return false
    }
    return !info.IsDir()
}

func dirExists(filename string) bool {
    info, err := os.Stat(filename)
    if os.IsNotExist(err) {
        return false
    }
    return info.IsDir()
}