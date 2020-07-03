package main

import (
    "bytes"
    "fmt"
    "net/http"
)

func stf_reserve( config *Config, udid string ) (bool) {
    token := config.Stf.AdminToken
    json := fmt.Sprintf(`{"serial":"%s"`,udid)
    url := fmt.Sprintf("https://%s/api/v1/user/devices", config.Stf.HostName )
    req, _ := http.NewRequest("POST", url, bytes.NewReader( []byte( json ) ) )
    req.Header.Set( "Authorization", "Bearer " + token )
    client := http.Client{}
    _, err := client.Do( req )
    if err != nil {
        return false
    }
    
    return true
}

func stf_release( config *Config, udid string ) (bool) {
    token := config.Stf.AdminToken
    url := fmt.Sprintf("https://%s/api/v1/user/devices/%s", config.Stf.HostName, udid)
    req, _ := http.NewRequest("DELETE", url, nil )
    req.Header.Set( "Authorization", "Bearer " + token )
    client := http.Client{}
    _, err := client.Do( req )
    if err != nil {
        return false
    }
    
    return true
}