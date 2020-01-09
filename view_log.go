package main

import (
    "bufio"
    "bytes"
    "flag"
    "fmt"
    "io"
    "io/ioutil"
    "encoding/json"
    "os"
    "strings"
    "github.com/fsnotify/fsnotify"
    log "github.com/sirupsen/logrus"
)

type Config struct {
    LogFile         string `json:"log_file"`
    LinesLogFile    string `json:"lines_log_file"`
}

func main() {
    var configFile = flag.String( "config", "config.json", "Config file path" )
    var findProc   = flag.String( "proc"  , ""           , "Process to view log of" )
    flag.Parse()
    
    config := read_config( *configFile )
  
    fileName := config.LinesLogFile
    
    if *findProc == "" {
        fmt.Println("specify a log to view / tail ( view_log -proc [proc] ):\n  wdaproxy\n  stf_device_ios\n  device_trigger\n  video_enabler\n  stf_provider\n  ffmpeg\n  wda\n  device_trigger\n")
        os.Exit( 0 )
    }
    
    if *findProc == "wda" {
        fileName = "bin/wda/req_log.json"
    }
    
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        log.Fatal(err)
    }
    
    fh, err := os.Open( fileName )
    if err != nil {
        panic(err)
    }
    defer fh.Close()
    
    size := fileSize( fh )
    //fh.Seek( size, io.SeekStart )
    
    scanner  := bufio.NewScanner( fh )
    for scanner.Scan() {
        checkLine( []byte( scanner.Text() ), *findProc )
    }
    
    err = watcher.Add( fileName )
    if err != nil {
        log.Fatal(err)
    }
    for {
        select {
            case event := <-watcher.Events:
                if event.Op & fsnotify.Write == fsnotify.Write {
                    //fmt.Println("modify")
                    newSize := fileSize( fh )
                    
                    newBytes := newSize - size
                    
                    if newBytes > 0 {
                        //fmt.Printf("  dif: %d\n", newBytes )
                        
                        //f.Seek(pos, io.SeekStart)
                        buf := make( []byte, newBytes )
                        fh.Read( buf )
                        //fmt.Printf("  \"%s\"\n", string( buf ) )
                        
                        checkLine( buf, *findProc )
                        
                        size = newSize
                    }
                }
        }
    }
}

func read_config( configPath string ) *Config {
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
    config := Config{}
    json.Unmarshal( jsonBytes, &config )
    return &config
}

func checkLine( data []byte, findProc string ) {
    var dat map[string]interface{}
    
    startJ := strings.Index( string(data), "{" )
    endJ := strings.LastIndex( string(data), "}" )
    
    part := string(data)[ startJ : (endJ + 1) ]
    
    decoder := json.NewDecoder( strings.NewReader( part ) )
    for {
        err := decoder.Decode( &dat )
        if err == io.EOF {
            break
        }
        if err != nil {
            panic(err)
        }
        
        if findProc == "wda" {
            //fmt.Println( part )
            typeif := dat["type"]
            if typeif != nil {
                typ := typeif.(string)
                //fmt.Println( typ )
                if typ == "req.start" {
                    if dat["body_in"] != nil {
                        inStr := dat["body_in"].(string)
                        
                        fmt.Printf("Req URI: %s\n", dat["uri"].(string) )
                        if inStr[:1] == "{" {
                            var prettyJSON bytes.Buffer
                            error := json.Indent(&prettyJSON, []byte( inStr ), "", "  ")
                            if error != nil {
                                fmt.Println( inStr )
                            } else {
                                fmt.Println( prettyJSON.String() )
                            }
    
                            //dec2 :=- json.NewDecoder( strings.NewReader( dat["body_in"].(string)  ) )
                            //err = dec2.Decode( &dat )
                        } else {
                            fmt.Println( inStr )
                        }
                    }
                } else if typ == "req.done" {
                    if dat["body_out"] != nil {
                        outStr := dat["body_out"].(string)
                        fmt.Printf("Response to URI: %s\n", dat["uri"].(string) )
                        
                        fmt.Println( outStr )
                    }
                }
            }
        } else {
            proc := dat["proc"].(string)
            if proc == findProc {
                //fmt.Println(dat)
                line := dat["line"].(string)
                fmt.Println( line )
            }
        }
    }
}

func fileSize( fh *os.File ) (int64) {
    newinfo, err := fh.Stat()
    if err != nil {
        panic(err)
    }
    return newinfo.Size()
}