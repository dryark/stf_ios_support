package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "os"
    
    log "github.com/sirupsen/logrus"
)

type Config struct {
    DeviceTrigger    string `json:"device_trigger"`
    VideoEnabler     string `json:"video_enabler"`
    MirrorFeedBin    string `json:"mirrorfeed_bin"`
    WDARoot          string `json:"wda_root"`
    CoordinatorPort  int    `json:"coordinator_port"`
    WDAProxyBin      string `json:"wdaproxy_bin"`
    WDAProxyPort     int    `json:"wdaproxy_port"`
    MirrorFeedPort   int    `json:"mirrorfeed_port"`
    Pipe             string `json:"pipe"`
    SkipVideo        bool   `json:"skip_video"`
    Ffmpeg           string `json:"ffmpeg"`
    STFIP            string `json:"stf_ip"`
    STFHostname      string `json:"stf_hostname"`
    WDAPorts         string `json:"wda_ports"`
    VidPorts         string `json:"vid_ports"`
    DevIosPorts      string `json:"dev_ios_ports"`
    DevIosPort       int    `json:"dev_ios_port"`
    LogFile          string `json:"log_file"`
    LinesLogFile     string `json:"lines_log_file"`
    VpnName          string `json:"vpn_name"`
    NetworkInterface string `json:"network_interface"`
    ConfigPath       string `json:"config_path"`
    RootPath         string `json:"root_path"`
    WDAWrapperStdout string `json:"wda_wrapper_stdout"`
    WDAWrapperStderr string `json:"wda_wrapper_stderr"`
    WDAWrapperBin    string `json:"wda_wrapper_bin"`
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
        config = Config{
            DeviceTrigger:   "bin/osx_ios_device_trigger",
            VideoEnabler:    "bin/osx_ios_video_enabler",
            WDAProxyBin:     "bin/wdaproxy",
            MirrorFeedBin:   "bin/mirrorfeed",
            WDARoot:         "./bin/wda",
            Ffmpeg:          "bin/ffmpeg",
            CoordinatorPort: 8027,
            MirrorFeedPort:  8000,
            WDAProxyPort:    8100,
            DevIosPort:      9240,
            DevIosPorts:     "9240-9250",
            Pipe:            "pipe",
            ConfigPath:      "",
            RootPath:        "",
            WDAWrapperStdout:"./logs/wda_wrapper_stdout",
            WDAWrapperStderr:"./logs/wda_wrapper_stderr",
            WDAWrapperBin:   "bin/wda_wrapper",
        }
        json.Unmarshal( jsonBytes, &config )
        if config.ConfigPath != "" {
            configPath = config.ConfigPath
            continue
        }
        break
    }
    return &config
}