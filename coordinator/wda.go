package main

import (
  "bytes"
  "fmt"
  "net/http"
  "strconv"
  "strings"
  "sync"
  "time"
  uj "github.com/nanoscopic/ujsonin/mod"
  log "github.com/sirupsen/logrus"
)

type WDAType struct {
  base string
  channel chan DevEvent
  devd *RunningDev
}

func NewWDACaller( base string ) ( *WDAType ) {
  self := WDAType { base: base }
  return &self
}

func NewTempWDA( o ProcOptions ) ( *WDAType ) {
  tempCh := make( chan DevEvent )
  wda := WDAType {
    channel: tempCh,
    base: ( "http://" + o.curIP + ":" + strconv.Itoa( o.devd.wdaPort ) ),
    devd: o.devd,
  }
  
  proc_wdaproxy( o, tempCh, true )
  
  // Wait for WDA to actually start up
  for {
    done := 0
    select {
      case devEvent := <- tempCh:
        if devEvent.action == 4 {
          log.Info("TempWDA Started")
          done = 1
          break
        }
    }
    if done == 1 { break }
  }
  
  return &wda
}

func aio_reset_media_services( o ProcOptions ) {
  baseCopy := *(o.baseProgs)
  o.baseProgs = &baseCopy
  devCopy := *(o.devd)
  o.devd = &devCopy
  devCopy.lock = &sync.Mutex{}
  
  wda := NewTempWDA( o )
  time.Sleep( time.Second * 2 )
  wda.reset_media_services()
  o.baseProgs.shuttingDown = true
  wda.end()
  time.Sleep( time.Second * 2 )
}

func ( self *WDAType ) end() {
  devd := self.devd
  wdaProc := devd.process["wdaproxytemp"]
  log.WithFields( log.Fields{
      "type": "proc_kill",
      "pid": wdaProc.pid,
  } ).Debug("Attempting to kill")
  wdaProc.Kill()
}

func ( self *WDAType ) reset_media_services() {
  sid := self.create_session( "com.apple.Preferences" )
  devEl := self.el_by_name( sid, "Developer" )
  log.Debug("Got ID " + devEl + " for Developer item" )
  self.scroll_to( sid, devEl )
  self.click( sid, devEl )
  resetEl := self.el_by_name( sid, "Reset Media Services" )
  log.Debug("Got ID " + resetEl + " for Reset Media Services item" )
  self.scroll_to( sid, resetEl )
  self.click( sid, resetEl )
  self.home( sid )
}

func ( self *WDAType ) el_by_name( sid string, name string ) ( string ) {
  json := fmt.Sprintf(`{
    "using": "name",
    "value": "%s"
  }`, name )
  url := self.base + "/session/" + sid + "/element"
  log.Info( "visiting " + url )
  resp, _ := http.Post( url, "application/json", strings.NewReader( json ) )
  res := resp_to_val( resp )
  return res.Get("ELEMENT").String()
}

func ( self *WDAType ) click( sid string, eid string ) {
  url := self.base + "/session/" + sid + "/element/" + eid + "/click"
  log.Info( "visiting " + url )
  resp, _ := http.Post( url, "application/json", strings.NewReader( "{}" ) )
  if resp.StatusCode != 200 {
    log.Error( "got resp" + strconv.Itoa( resp.StatusCode ) + "from " + url )
  }
  //res := resp_to_val( resp )
}

func ( self *WDAType ) scroll_to( sid string, eid string ) {
  url := self.base + "/session/" + sid + "/wda/element/" + eid + "/scroll"
  log.Info( "visiting " + url )
  resp, _ := http.Post( url, "application/json", strings.NewReader( "{\"toVisible\":1}" ) )
  if resp.StatusCode != 200 {
    log.Error( "got resp" + strconv.Itoa( resp.StatusCode ) + "from " + url )
  }
}

func ( self *WDAType ) home( sid string ) {
  url := self.base + "/wda/homescreen"
  log.Info( "visiting " + url )
  resp, _ := http.Post( url, "application/json", strings.NewReader( "{}" ) )
  if resp.StatusCode != 200 {
    log.Error( "got resp" + strconv.Itoa( resp.StatusCode ) + "from " + url )
  }
}

func ( self *WDAType ) create_session( bundle string ) ( string ) {
  ops := fmt.Sprintf( `{
    "capabilities": {
      "alwaysMatch": {},
      "firstMatch": [
        {
          "arguments": [],
          "bundleId": "%s",
          "environment": {},
          "shouldUseSingletonTestManager": true,
          "shouldUseTestManagerForVisibilityDetection": false,
          "shouldWaitForQuiescence": true
        }
      ]
    }
  }`, bundle );
  resp, _ := http.Post( self.base + "/session", "application/json", strings.NewReader( ops ) )
  res := resp_to_val( resp )
  return res.Get("sessionId").String()
}

func wda_session( base string ) ( string ) {
  resp, _ := http.Get( base + "/status" )
  content, _ := uj.Parse( []byte( resp_to_str( resp ) ) )
  sid := content.Get("sessionId").String()
  return sid;
}

func ( self *WDAType ) is_locked() ( bool ) {
  resp, _ := http.Get( self.base + "/wda/locked" )
  respStr := resp_to_str( resp )
  fmt.Printf("response str:%s\n", respStr)
  content, _ := uj.Parse( []byte( respStr ) )
  //fmt.Printf("output:%s\n", content )
  return content.Get("value").Bool()
}

func ( self *WDAType ) unlock() {
  http.Post( self.base + "/wda/unlock", "application/json", strings.NewReader( "{}" ) )
}

func source( base string ) ( string ) {
  resp, _ := http.Get( base + "/source" )
  res := resp_to_str( resp )
  //print Dumper( res )
  return res
}

func wda_apps_list( base string ) ( string ) {
  sid := wda_session( base )
  resp, _ := http.Get( base + "/session/" + sid + "/wda/apps/list" )
  res := resp_to_str( resp )
  //print Dumper( res )
  return res
}

func wda_battery_info( base string ) ( string ) {
  sid := wda_session( base )
  resp, _ := http.Get( base + "/session/" + sid + "/wda/batteryInfo" )
  res := resp_to_str( resp )
  //print Dumper( $res )
  return res
}

func resp_to_str( resp *http.Response ) ( string ) {
  body := resp.Body
  buf := new( bytes.Buffer )
  buf.ReadFrom( body )
  return buf.String()  
}

func resp_to_val( resp *http.Response ) ( *uj.JNode ) {
  rawContent := resp_to_str( resp )
  if !strings.HasPrefix( rawContent, "{" ) {
    return nil // &JNode{ nodeType: 1, hash: NewNodeHash() }
  }
  content, _ := uj.Parse( []byte( rawContent ) )
  val := content.Get("value")
  if val == nil { return content }
  return val
}