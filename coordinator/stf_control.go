package main

import (
    "bytes"
    "crypto/tls"
    "fmt"
    "io/ioutil"
    "net/http"
    "strings"
)

func stf_reserve( config *Config, udid string ) (bool) {
    tr := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
    }
    
    token := config.Stf.AdminToken
    json := fmt.Sprintf(`{"serial":"%s"}`,udid)
    fmt.Println("Sending:",json)
    url := fmt.Sprintf("https://%s/api/v1/user/devices", config.Stf.HostName )
    req, _ := http.NewRequest("POST", url, bytes.NewReader( []byte( json ) ) )
    req.Header.Set( "Authorization", "Bearer " + token )
    req.Header.Set( "Content-Type", "application/json" )
    client := http.Client{ Transport: tr }
    resp, err := client.Do( req )
    if err != nil || !strings.HasPrefix( resp.Status, "200" ) {
        fmt.Println("Error:", err )
        fmt.Println("Response Status:", resp.Status)
        body, _ := ioutil.ReadAll(resp.Body)
        fmt.Println("Response Body:", string(body))
    
        return false
    }
    
    return true
}

func stf_release( config *Config, udid string ) (bool) {
    tr := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
    }
    
    token := config.Stf.AdminToken
    url := fmt.Sprintf("https://%s/api/v1/user/devices/%s", config.Stf.HostName, udid)
    req, _ := http.NewRequest("DELETE", url, nil )
    req.Header.Set( "Authorization", "Bearer " + token )
    client := http.Client{ Transport: tr }
    resp, err := client.Do( req )
    if err != nil || !strings.HasPrefix( resp.Status, "200" ) {
        fmt.Println("Error:", err )
        fmt.Println("Response Status:", resp.Status)
        body, _ := ioutil.ReadAll(resp.Body)
        fmt.Println("Response Body:", string(body))
        
        return false
    }
    
    return true
}