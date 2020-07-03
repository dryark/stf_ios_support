package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "os"
    
    log "github.com/sirupsen/logrus"
)

type Config struct {
    WdaFolder  string        `json:"wda_folder"`
    Network    NetConfig     `json:"network"`
    Stf        STFConfig     `json:"stf"`
    Video      VideoConfig   `json:"video"`
    FrameServer FrameServerConfig `json:"frameserver"`
    Install    InstallConfig `json:"install"`
    Log        LogConfig     `json:"log"`
    BinPaths   BinPathConfig `json:"bin_paths"`
    Vpn        VPNConfig     `json:"vpn"`
    Timing     TimingConfig  `json:"timing"`
    ConfigPath string        `json:"config_path"`
    // The following are only used internally
    WDAProxyPort   int
    MirrorFeedPort int
    DevIosPort     int
    VncPort        int
    DecodeInPort   int
    DecodeOutPort  int
}

type NetConfig struct {
    Coordinator int    `json:"coordinator_port"`
    Mirrorfeed  int    `json:"mirrorfeed_port"`
    Video       string `json:"video_ports"`
    DevIos      string `json:"dev_ios_ports"`
    Vnc         string `json:"vnc_ports"`
    Wda         string `json:"proxy_ports"`
    Decode      string `json:"decode_ports"`
    Iface       string `json:"interface"`
}

type STFConfig struct {
    Ip       string `json:"ip"`
    HostName string `json:"hostname"`
    Location string `json:"location"`
    AdminToken string `json:"admin_token"`
}

type VideoConfig struct {
    Enabled     bool   `json:"enabled"`
    Method      string `json:"method"`
    UseVnc      bool   `json:"use_vnc"`
    VncScale    int    `json:"vnc_scale"`
    VncPassword string `json:"vnc_password"`
    FrameRate   int    `json:"frame_rate"`
}

type InstallConfig struct {
    RootPath   string `json:"root_path"`
    ConfigPath string `json:"config_path"`
}

type LogConfig struct {
    Main             string `json:"main"`
    MainApp          string `json:"main_app"`
    ProcLines        string `json:"proc_lines"`
    WDAWrapperStdout string `json:"wda_wrapper_stdout"`
    WDAWrapperStderr string `json:"wda_wrapper_stderr"`
    OpenVPN          string `json:"openvpn"`
}

type BinPathConfig struct {
    WdaProxy      string `json:"wdaproxy"`
    DeviceTrigger string `json:"device_trigger"`
    IosVideoStream string `json:"ios_video_stream"`
    IosVideoPull   string `json:"ios_video_pull"`
    H264ToJpeg    string `json:"h264_to_jpeg"`
    Openvpn       string `json:"openvpn"`
    Iproxy        string `json:"iproxy"`
    WdaWrapper    string `json:"wdawrapper"`
    IVF           string `json:"ivf"`
}

type VPNConfig struct {
    VpnType    string `json:"type"`
    TblickName string `json:"tblick_name"`
    OvpnWd     string `json:"ovpn_working_dir"`
    OvpnConfig string `json:"ovpn_config"`
}

type FrameServerConfig struct {
    Secure bool `json:"secure"`
    Cert string `json:"cert"`
    Key string `json:"key"`
    Width int `json:"width"`
    Height int `json:"height"`
}

type TimingConfig struct {
    WdaRestart int `json:"wda_restart"`
}

func read_config( configPath string ) *Config {
    var config Config
    
    for {
        fh, serr := os.Stat( configPath )
        if serr != nil {
            log.WithFields( log.Fields{
                "type": "err_read_config",
                "error": serr,
                "config_path": configPath,
            } ).Fatal("Could not read specified config path")
        }
        
        var configFile string
        switch mode := fh.Mode(); {
            case mode.IsDir():
                configFile = fmt.Sprintf("%s/config.json", configPath)
            case mode.IsRegular():
                configFile = configPath
        }
    
        configFh, err := os.Open( configFile )
        if err != nil {
            log.WithFields( log.Fields{
                "type": "err_read_config",
                "config_file": configFile,
                "error": err,
            } ).Fatal("failed reading config file")
        }
        defer configFh.Close()
    
        jsonBytes, _ := ioutil.ReadAll( configFh )
        
        defaultJson := `{
          "wda_folder": "./bin/wda",
          "xcode_dev_team_id": "",
          "network": {
            "coordinator_port": 8027,
            "video_ports":     "8000-8005",
            "dev_ios_ports":   "9240-9250",
            "vnc_ports":       "5901-5911",
            "proxy_ports":     "8100-8105",
            "decode_ports":    "7878-7888",
            "interface": "en0"
          },
          "stf":{
            "ip": "",
            "hostname": "",
            "location": "",
            "admin_token": "",
          },
          "video":{
            "enabled": true,
            "method": "avfoundation",
            "use_vnc": false,
            "vnc_scale": 2,
            "vnc_password": "",
            "frame_rate": 5
          },
          "frameserver":{
            "secure": false,
            "cert": "",
            "key": "",
            "width": 0,
            "height": 0
          },
          "install":{
            "root_path": "",
            "config_path": ""
          },
          "log":{
            "main":               "logs/coordinator",
            "main_app":           "logs/app",
            "proc_lines":         "logs/procs",
            "wda_wrapper_stdout": "./logs/wda_wrapper_stdout",
            "wda_wrapper_stderr": "./logs/wda_wrapper_stderr",
            "openvpn":            "logs/openvpn.log"
          },
          "vpn":{
            "type":             "none",
            "ovpn_working_dir": "/usr/local/etc/openvpn",
            "tblick_name":      ""
          },
          "bin_paths":{
            "wdaproxy":       "bin/wdaproxy",
            "device_trigger": "bin/osx_ios_device_trigger",
            "openvpn":        "/usr/local/opt/openvpn/sbin/openvpn",
            "iproxy":         "/usr/local/bin/iproxy",
            "wdawrapper":     "bin/wda_wrapper",
            "ios_video_stream":"bin/ios_video_stream",
            "ios_video_pull":"bin/ios_video_pull",
            "h264_to_jpeg":   "bin/decode",
            "ivf":            "bin/ivf_pull"
          },
          "repos":{
            "stf": "https://github.com/nanoscopic/stf-ios-provider.git",
            "wda": "https://github.com/nanoscopic/WebDriverAgent.git"
          },
          "timing":{
            "wda_restart": 240
          }
        }`
        
        config = Config{
          MirrorFeedPort:  8000,
          WDAProxyPort:    8100,
          DevIosPort:      9240,
          VncPort:         5901,
          DecodeOutPort:   7878,
          DecodeInPort:    7879,
        }
        
        err = json.Unmarshal( []byte( defaultJson ), &config )
        if err != nil {
          log.Fatal( "1 ", err )
        }
        
        err = json.Unmarshal( jsonBytes, &config )
        if err != nil {
          log.Fatal( "2 ", err )
        }
        
        //jsonCombined, _ := json.MarshalIndent(config, "", "  ")
        //fmt.Printf("Combined config:%s\n", string( jsonCombined ) )
        //os.Exit(0)
        
        if config.ConfigPath != "" {
            configPath = config.ConfigPath
            continue
        }
        break
    }
    return &config
}