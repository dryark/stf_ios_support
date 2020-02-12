package main

import (
  "bytes"
  "encoding/json"
  "fmt"
  "net/http"
  "text/template"
  
  log "github.com/sirupsen/logrus"
)

func coro_http_server( config *Config, devEventCh chan<- DevEvent, baseProgs *BaseProgs, runningDevs map [string] *RunningDev, lineTracker *InMemTracker ) {
    // start web server waiting for trigger http command for device connect and disconnect
    var listen_addr = fmt.Sprintf( "0.0.0.0:%d", config.Network.Coordinator )
    go startServer( devEventCh, listen_addr, baseProgs, runningDevs, lineTracker )
}

func startServer( devEventCh chan<- DevEvent, listen_addr string, baseProgs *BaseProgs, runningDevs map[string] *RunningDev, lineTracker *InMemTracker ) {
    log.WithFields( log.Fields{
        "type": "http_start",
    } ).Debug("HTTP server started")

    rootClosure := func( w http.ResponseWriter, r *http.Request ) {
        handleRoot( w, r, baseProgs, runningDevs )
    }
    devinfoClosure := func( w http.ResponseWriter, r *http.Request ) {
        reqDevInfo( w, r, baseProgs, runningDevs )
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
    http.HandleFunc( "/dev_connect", connectClosure )
    http.HandleFunc( "/dev_disconnect", disconnectClosure )
    http.HandleFunc( "/new_interface", ifaceClosure )
    http.HandleFunc( "/log", logClosure )
    log.Fatal( http.ListenAndServe( listen_addr, nil ) )
}

func fixUuid( uuid string ) (string) {
    if len(uuid) == 24 {
        p1 := uuid[:8]
        p2 := uuid[8:]
        uuid = fmt.Sprintf("%s-%s",p1,p2)
    }
    return uuid
}

func reqDevInfo( w http.ResponseWriter, r *http.Request, baseProgs *BaseProgs, runningDevs map[string] *RunningDev ) {
    query := r.URL.Query()
    uuid := query.Get("uuid")
    info := getAllDeviceInfo( uuid )

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
    if baseProgs.trigger == nil { device_trigger = "off" }
    video_enabler := "<font color='green'>on</font>"
    if baseProgs.vidEnabler == nil { video_enabler = "off" }
    stf := "<font color='green'>on</font>"
    if baseProgs.stf == nil { stf = "off" }

    devOut := ""
    for _, dev := range runningDevs {
        mirror := "<font color='green'>on</font>"
        if dev.mirror == nil { mirror = "off" }

        ff := "<font color='green'>on</font>"
        if dev.ff == nil { ff = "off" }

        //proxy := "<font color='green'>on</font>"
        //if dev.proxy == nil { proxy = "off" }

        device := "<font color='green'>on</font>"
        if dev.device == nil { device = "off" }

        var str bytes.Buffer
        deviceTpl.Execute( &str, map[string] string {
            "uuid":   "<a href='/devinfo?uuid=" + dev.uuid + "'>" + dev.uuid + "</a>",
            "name":   dev.name,
            "mirror": mirror,
            "ff":     ff,
            //"proxy":  proxy,
            "device": device,
        } )
        devOut += str.String()
    }

    rootTpl.Execute( w, map[string] string{
        "device_trigger": device_trigger,
        "video_enabler":  video_enabler,
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
        "type": "iface_body",
        //"body": body.String(),
        //"struct": ifaceData,
        "uuid": ifaceData.Serial,
        "class": ifaceData.Class,
        "subclass": ifaceData.SubClass,
    } ).Info("New Interface")
    
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
    <td>Video Mirror</td>
    <td>{{.mirror}}</td>
  </tr>
  <tr>
    <td>FFMpeg</td>
    <td>{{.ff}}</td>
  </tr>
  <tr>
    <td>WDA Proxy</td>
    <td>{{.proxy}}</td>
  </tr>
  <tr>
    <td>STF Device-IOS Unit</td>
    <td>{{.device}}</td>
  </tr>
</table>
`))

var rootTpl = template.Must(template.New("root").Parse(`
<!DOCTYPE html>
<html>
	<head>
	</head>
	<body>
	Base Processes:
	<table border=1 cellpadding=5 cellspacing=0>
	  <tr>
	    <td>Device Trigger</td>
	    <td>{{.device_trigger}}</td>
	  </tr>
	  <tr>
	    <td>Video Enabler</td>
	    <td>{{.video_enabler}}</td>
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
