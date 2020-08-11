package main

import (
    "bytes"
    "crypto/tls"
    "fmt"
    "io/ioutil"
    "net/http"
    "strings"
)

func NewStfClient() (http.Client) {
    tr := &http.Transport{
        TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
    }
    return http.Client{ Transport: tr }
}

func stf_set_auth( config *Config, req *http.Request ) {
    token := config.Stf.AdminToken
    req.Header.Set( "Authorization", "Bearer " + token )
}

func stf_do_request( config *Config, req *http.Request ) (bool, *http.Response) {
    client := NewStfClient()
    stf_set_auth( config, req )
    resp, err := client.Do( req )
    if err != nil || !strings.HasPrefix( resp.Status, "200" ) {
        fmt.Println("Error:", err )
        fmt.Println("Response Status:", resp.Status)
        body, _ := ioutil.ReadAll(resp.Body)
        fmt.Println("Response Body:", string(body))
    
        return false,nil
    }
    return true, resp
}

func stf_reserve( config *Config, udid string ) (bool) {
    json := fmt.Sprintf(`{"serial":"%s"}`,udid)
    fmt.Println("Sending:",json)
    url := fmt.Sprintf("https://%s/api/v1/user/devices", config.Stf.HostName )
    req, _ := http.NewRequest("POST", url, bytes.NewReader( []byte( json ) ) )
    req.Header.Set( "Content-Type", "application/json" )
    
    success, _ := stf_do_request( config, req )
    return success
}

func stf_release( config *Config, udid string ) (bool) {
    url := fmt.Sprintf("https://%s/api/v1/user/devices/%s", config.Stf.HostName, udid)
    req, _ := http.NewRequest("DELETE", url, nil )
    
    success, _ := stf_do_request( config, req )
    return success
}