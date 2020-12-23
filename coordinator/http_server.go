package main

import (
  "bytes"
  "encoding/json"
  "fmt"
  "net/http"
  "strconv"
  "strings"
  "text/template"
  
  log "github.com/sirupsen/logrus"
  uj "github.com/nanoscopic/ujsonin/mod"
)

func coro_http_server( config *Config, devEventCh chan<- DevEvent, baseProgs *BaseProgs, runningDevs map [string] *RunningDev, lineTracker *InMemTracker ) {
    // start web server waiting for trigger http command for device connect and disconnect
    var listen_addr = fmt.Sprintf( "0.0.0.0:%d", config.Network.Coordinator )
    go startServer( config, devEventCh, listen_addr, baseProgs, runningDevs, lineTracker )
}

func coro_mini_http_server( config *Config, devEventCh chan<- DevEvent, devd *RunningDev ) {
    var listen_addr = fmt.Sprintf( "0.0.0.0:%d", config.Network.Coordinator )
    go startMiniServer( devEventCh, devd, listen_addr )
}

func startServer( config *Config, devEventCh chan<- DevEvent, listen_addr string, baseProgs *BaseProgs, runningDevs map[string] *RunningDev, lineTracker *InMemTracker ) {
    log.WithFields( log.Fields{
        "type": "http_start",
    } ).Debug("HTTP server started")

    rootClosure := func( w http.ResponseWriter, r *http.Request ) {
        handleRoot( w, r, baseProgs, runningDevs )
    }
    devinfoClosure := func( w http.ResponseWriter, r *http.Request ) {
        reqDevInfo( config, w, r, baseProgs, runningDevs )
    }
    logClosure := func( w http.ResponseWriter, r *http.Request ) {
        handleLog( w, r, baseProgs, runningDevs, lineTracker )
    }
    http.HandleFunc( "/devinfo", devinfoClosure )
    http.HandleFunc( "/", rootClosure )
    connectClosure := func( w http.ResponseWriter, r *http.Request ) {
        deviceConnect( w, r, devEventCh )
    }
    disconnectClosure := func( w http.ResponseWriter, r *http.Request ) {
        deviceDisconnect( w, r, devEventCh )
    }
    ifaceClosure := func( w http.ResponseWriter, r *http.Request ) {
        newInterface( w, r, devEventCh )
    }
    frameClosure := func( w http.ResponseWriter, r *http.Request ) {
        handleFrame( w, r, devEventCh )
    }
    procRestartClosure := func( w http.ResponseWriter, r *http.Request ) {
        handleProcRestart( w, r, runningDevs )
    }
    http.HandleFunc( "/dev_connect", connectClosure )
    http.HandleFunc( "/dev_disconnect", disconnectClosure )
    http.HandleFunc( "/new_interface", ifaceClosure )
    http.HandleFunc( "/frame", frameClosure )
    http.HandleFunc( "/log", logClosure )
    http.HandleFunc( "/procrestart", procRestartClosure )
    err := http.ListenAndServe( listen_addr, nil )
    log.WithFields( log.Fields{
        "type": "http_server_fail",
        "error": err,
    } ).Debug("HTTP ListenAndServe Error")
}

func startMiniServer( devEventCh chan<- DevEvent, devd *RunningDev, listen_addr string ) {
    frameClosure := func( w http.ResponseWriter, r *http.Request ) {
        handleFrame( w, r, devEventCh )
    }
    http.HandleFunc( "/frame", frameClosure )
    err := http.ListenAndServe( listen_addr, nil )
    log.WithFields( log.Fields{
        "type": "http_server_fail",
        "error": err,
    } ).Debug("HTTP ListenAndServe Error")
}

func fixUuid( uuid string ) (string) {
    if len(uuid) == 24 {
        p1 := uuid[:8]
        p2 := uuid[8:]
        uuid = fmt.Sprintf("%s-%s",p1,p2)
    }
    return uuid
}

func reqDevInfo( config *Config, w http.ResponseWriter, r *http.Request, baseProgs *BaseProgs, runningDevs map[string] *RunningDev ) {
    query := r.URL.Query()
    uuid := query.Get("uuid")
    info := getAllDeviceInfo( config, uuid )

    names := map[string] string {
        "DeviceName":      "Device Name",
        "EthernetAddress": "MAC",
        "ModelNumber":     "Model",
        //"HardwareModel": "Hardware Model",
        "PhoneNumber":     "Phone Number",
        "ProductType":     "Product",
        "ProductVersion":  "IOS Version",
        "UniqueDeviceID":  "Wifi MAC",
        "InternationalCircuitCardIdentity":      "ICCI",
        "InternationalMobileEquipmentIdentity":  "IMEI",
        "InternationalMobileSubscriberIdentity": "IMSI",
    }

    for key, descr := range names {
        val := info[key]
        fmt.Fprintf( w, "%s: %s<br>\n", descr, val )
    }
}

func handleRoot( w http.ResponseWriter, r *http.Request, baseProgs *BaseProgs, runningDevs map[string] *RunningDev ) {
    device_trigger := "<font color='green'>on</font>"
    if baseProgs.process["device_trigger"] == nil { device_trigger = "off" }
    
    stf := "<font color='green'>on</font>"
    stfProc := baseProgs.process["stf_ios_provider"]
    if stfProc == nil || stfProc.cmd == nil { stf = "off" }

    devOut := ""
    for _, dev := range runningDevs {
        ivsProc := dev.process["ios_video_stream"]
        ios_video_stream := "<font color='green'>on</font>"
        if ivsProc == nil || ivsProc.cmd == nil { ios_video_stream = "off" }

        wda := "<font color='green'>up</font>"
        
        proxyProc := dev.process["wdaproxy"]
        proxy := "<font color='green'>on</font>"
        if proxyProc == nil || proxyProc.cmd == nil {
            proxy = "off"
            wda = "down"
        } else {
            if dev.wda == false {
                wda = "starting"
            }
        }
        
        devUnitProc := dev.process["stf_device_ios"]
        device := "<font color='green'>on</font>"
        if devUnitProc == nil || devUnitProc.cmd == nil { device = "off" }

        ivfProc := dev.process["ivf"]
        ivf_pull := "<font color='green'>on</font>"
        if ivfProc == nil || ivfProc.cmd == nil { ivf_pull = "off" }
        
        owner := dev.owner
        if owner == "" { owner = "none" }
        
        var str bytes.Buffer
        deviceTpl.Execute( &str, map[string] string {
            "uuid":             "<a href='/devinfo?uuid=" + dev.uuid + "'>" + dev.uuid + "</a>",
            "rawuuid":          dev.uuid,
            "name":             dev.name,
            "ios_video_stream": ios_video_stream,
            "proxy":            proxy,
            "deviceunit":       device,
            "owner":            owner,
            "wda":              wda,
            "ivf_pull":         ivf_pull,
            "vid_port":         strconv.Itoa( dev.vidPort ),
            "vid_stream_port":  strconv.Itoa( dev.streamPort ),
        } )
        devOut += str.String()
    }

    rootTpl.Execute( w, map[string] string{
        "device_trigger": device_trigger,
        "stf":            stf,
        "devices":        devOut,
    } )
}

func handleLog( w http.ResponseWriter, r *http.Request, baseProgs *BaseProgs, runningDevs map[string] *RunningDev, lineTracker *InMemTracker ) {
    procTracker := lineTracker.procTrackers["stf_device_ios"]
    
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    
    que := procTracker.que
    item := que.Back()
    for {
        json := item.Value.(string)
        fmt.Fprintf( w, "%s<br>\n",json )
        item = item.Prev()
        if item == nil {
            break
        }
    }
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

type IFaceData struct {
    Serial   string `json:"uuid"`
    Class    string `json:"class"`
    SubClass string `json:"subclass"`
    Vendor   string `json:"vendor"`
    Product  string `json:"product"`
}

func newInterface( w http.ResponseWriter, r *http.Request, devEventCh chan<- DevEvent ) {
    //snprintf( postdata, 255, "{\"class\":\"%02x\",\"subclass\":\"%02x\",\"vendor\":\"%04x\",\"product\":\"%04x\"}", cls, subcls, vendor, product );
    ifaceData := IFaceData{}
    
    body := new(bytes.Buffer)
    body.ReadFrom(r.Body)
    json.Unmarshal(body.Bytes(), &ifaceData )
    
    log.WithFields( log.Fields{
        "type":     "iface_body",
        //"body":   body.String(),
        //"struct": ifaceData,
        "uuid":     ifaceData.Serial,
        "class":    ifaceData.Class,
        "subclass": ifaceData.SubClass,
    } ).Debug("New Interface")
    
    if ifaceData.Class == "ff" && ifaceData.SubClass == "2a" {
        // signal device loop of video interface addition
        devEvent := DevEvent{}
        devEvent.action = 2
        r.ParseForm()
        uuid := fixUuid( ifaceData.Serial )
        devEvent.uuid = uuid
        devEventCh <- devEvent
    }
}

func handleProcRestart( w http.ResponseWriter, r *http.Request, runningDevs map[string] *RunningDev ) {
    body := new(bytes.Buffer)
    body.ReadFrom(r.Body)
    root, _ := uj.Parse( body.Bytes() )
    
    uuid := root.Get("uuid").String()
    proc := root.Get("proc").String()
    
    onRelease := false
    onReleaseNode := root.Get("onRelease")
    if onReleaseNode != nil { onRelease = true }
    
    devd := runningDevs[ uuid ]
    if proc == "wdaproxy" {
        restart_wdaproxy( devd, onRelease )
    } else if proc == "stf_device_ios" {
        restart_device_unit( devd )
    } else if proc == "ivf" {
        restart_ivf( devd )
    } else if proc == "ios_video_stream" {
        restart_ios_video_stream( devd )
    }
}

func handleFrame( w http.ResponseWriter, r *http.Request, devEventCh chan<- DevEvent ) {
    body := new(bytes.Buffer)
    body.ReadFrom(r.Body)
    str := string(body.Bytes())
    i := strings.Index( str, "}" )
    fmt.Printf("String to parse:%s\n", str[:i] )
    root, _ := uj.Parse( body.Bytes() )
    
    msgType := root.Get("type").String()
    
    if msgType == "frame1" {
        width := root.Get("width").Int()
        height := root.Get("height").Int()
        clickScale := root.Get("clickScale").Int()
        uuid := root.Get("uuid").String()
        devEvent := DevEvent{
            action: 3,
            width: width,
            height: height,
            clickScale: clickScale,
            uuid: uuid,
        }
        
        devEventCh <- devEvent
    } 
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
    <td>Owner</td>
    <td>{{.owner}}</td>
  </tr>
  <tr>
    <td>IOS Video Stream</td>
    <td>{{.ios_video_stream}} ports:{{.vid_port}},{{.vid_stream_port}}</td>
    <td><button onclick="vidstream_restart('{{.rawuuid}}')">Restart</button>
  </tr>
  <tr>
    <td>IOS AVFoundation Frame Pull ( ivf_pull )</td>
    <td>{{.ivf_pull}}</td>
    <td><button onclick="ivf_restart('{{.rawuuid}}')">Restart</button>
  </tr>
  <tr>
    <td>WDA Proxy</td>
    <td>{{.proxy}}</td>
    <td>
      <button id='wdabtn' onclick="wda_restart('{{.rawuuid}}')">Restart</button><br>
      <button id='wdabtn' onclick="wda_restart_on_release('{{.rawuuid}}')">Restart on Release</button>
    </td>
  </tr>
  <tr>
    <td>WDA</td>
    <td>{{.wda}}</td>
  </tr>
  <tr>
    <td>STF Device-IOS Unit</td>
    <td>{{.deviceunit}}</td>
    <td><button id='dubtn' onclick="devunit_restart('{{.rawuuid}}')">Restart</button>
  </tr>
</table>
`))

var rootTpl = template.Must(template.New("root").Parse(`
<!DOCTYPE html>
<html>
	<head>
	  <script>
	    function getel( id ) {
        return document.getElementById( id );
      }
      function req( type, url, handler, body ) {
        var xhr = new XMLHttpRequest();
        xhr.open( type, url );
        xhr.responseType = 'json';
        xhr.onload = function(x) { handler(x,xhr); }
        if( type == 'POST' && body ) xhr.send(body);
        else xhr.send();
      }
      function clickAt( pos ) {
        req( 'POST', 'http://localhost:8100/session/' + session + '/wda/tap/0', function() {}, JSON.stringify( { x: pos[0]/(1080/2)*wid, y: pos[1]/(1920/2)*heg } ) );
      }
      window.addEventListener("load", function(evt) {
      } );
      function wda_restart( uuid ) {
        req( 'POST', '/procrestart', function() {}, JSON.stringify( { uuid: uuid, proc:'wdaproxy' } ) );
      }
      function wda_restart_on_release( uuid ) {
        req( 'POST', '/procrestart', function() {}, JSON.stringify( { uuid: uuid, proc:'wdaproxy', onRelease: 1 } ) );
      }
      function devunit_restart( uuid ) {
        req( 'POST', '/procrestart', function() {}, JSON.stringify( { uuid: uuid, proc:'stf_device_ios' } ) );
      }
      function ivf_restart( uuid ) {
        req( 'POST', '/procrestart', function() {}, JSON.stringify( { uuid: uuid, proc:'ivf' } ) );
      }
      function vidstream_restart( uuid ) {
        req( 'POST', '/procrestart', function() {}, JSON.stringify( { uuid: uuid, proc:'ios_video_stream' } ) );
      }
	  </script>
	</head>
	<body>
	Base Processes:
	<table border=1 cellpadding=5 cellspacing=0>
	  <tr>
	    <td>Device Trigger</td>
	    <td>{{.device_trigger}}</td>
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
