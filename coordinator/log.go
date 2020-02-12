package main

import (
  "context"
  "fmt"
  "os"
  "os/signal"
  "strings"
  "syscall"
  "sync"
  "container/list"
  log "github.com/sirupsen/logrus"
)

type ProcTracker struct {
    que *list.List
    length int
}

type InMemTracker struct {
    procTrackers map[string] *ProcTracker
}

type JSONLog struct {
	  file      *os.File
	  fileName  string
	  formatter *log.JSONFormatter
	  failed    bool
	  hupData   *HupData
	  id        int
	  inMemTracker *InMemTracker
	  mutex     sync.Mutex
}

type HupData struct {
    hupA bool
    hupB bool
    mutex sync.Mutex
}

func ( hook *JSONLog ) Fire( entry *log.Entry ) error {
    // If we have failed to write to the file; don't bother trying
    if hook.failed { return nil }

    jsonformat, _ := hook.formatter.Format( entry )

    fh := hook.file

    doHup := false
    hupData := hook.hupData
    hupData.mutex.Lock()
    if hook.id == 1 {
        doHup = hupData.hupA
        if doHup { hupData.hupA = false }
    } else if hook.id == 2 {
        doHup = hupData.hupB
        if doHup { hupData.hupB = false }
    }
    hupData.mutex.Unlock()

    if doHup {
        fh.Close()
        fhnew, err := os.OpenFile( hook.fileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666 )
        if err != nil {
            fmt.Fprintf( os.Stderr, "Unable to open file for writing: %v", err )
            fh = nil
        }
        fh = fhnew
        hook.file = fh

        log.WithFields( log.Fields{
            "type": "sighup",
            "state": "reopen",
            "file": hook.fileName,
        } ).Info("HUP requested")
        //fmt.Fprintf( os.Stdout, "Hup %s\n", hook.fileName )
    }

    var err error
    if entry.Context != nil {
        // There is context; this is meant for the lines logfile
        str := string( jsonformat )
        str = strings.Replace( str, "\"level\":\"info\",", "", 1 )
        str = strings.Replace( str, "\"msg\":\"\",", "", 1 )
        _, err = fh.WriteString( str )
        
        // Possibly better to us a coroutine to accept new log messages
        //   rather than lock on every one.
        hook.mutex.Lock()
        hook.inMemTracker.addEntry( entry, str )
        hook.mutex.Unlock()
    } else {
        _, err = fh.WriteString( string( jsonformat ) )
    }

    if err != nil {
        hook.failed = true
        fmt.Fprintf( os.Stderr, "Cannot write to logfile: %v", err )
        return err
    }

    return nil
}
func (tracker *InMemTracker) addEntry( entry *log.Entry, json string ) {
    if proc, ok := entry.Data["proc"]; ok {
        procS := proc.(string)
        var ok2 bool
        var pt *ProcTracker
        if pt, ok2 = tracker.procTrackers[ procS ]; !ok2 {
            pt = &ProcTracker{
                que: list.New(),
                length: 0,
            }
            tracker.procTrackers[ procS ] = pt
        }
        
        pt.length = pt.length + 1
        pt.que.PushBack( json )
        
        // Max out at 20 elements per queue
        if pt.length > 20 {
            e := pt.que.Front()
            pt.que.Remove(e)
        }
    }
}
func (hook *JSONLog) Levels() []log.Level {
    return []log.Level{ log.PanicLevel, log.FatalLevel, log.ErrorLevel, log.WarnLevel, log.InfoLevel, log.DebugLevel }
}
func AddJSONLog( logger *log.Logger, fileName string, id int, hupData *HupData ) ( *JSONLog ) {
    logFile, err := os.OpenFile( fileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0666 )
    if err != nil {
        fmt.Fprintf( os.Stderr, "Unable to open file for writing: %v", err )
    }

    fileHook := JSONLog{
        file:      logFile,
        fileName:  fileName,
        formatter: &log.JSONFormatter{},
        failed:    false,
        hupData:   hupData,
        id:        id,
        inMemTracker: NewInMemTracker(),
    }

    if logger == nil {
        log.AddHook( &fileHook )
    } else {
        logger.AddHook( &fileHook )
    }
    return &fileHook
}
func NewInMemTracker() ( *InMemTracker ) {
    newt := InMemTracker{
        procTrackers: make( map [string] *ProcTracker ),
    }
    return &newt
}
type DummyWriter struct {
}
func (self *DummyWriter) Write( p[]byte) (n int, err error) {
    return len(p), nil
}

func setup_log( config *Config, debug bool, jsonLog bool ) (*log.Entry, *InMemTracker) {
    if jsonLog {
        log.SetFormatter( &log.JSONFormatter{} )
    }

    lineLogger1 := log.New()
    dummyWriter := DummyWriter{}
    lineLogger1.SetOutput( &dummyWriter )
    lineLogger := lineLogger1.WithContext( context.Background() )

    if debug {
        log.WithFields( log.Fields{ "type": "debug_status" } ).Warn("Debugging enabled")
        log.SetLevel( log.DebugLevel )
        lineLogger1.SetLevel( log.DebugLevel )
    } else {
        log.SetLevel( log.InfoLevel )
        lineLogger1.SetLevel( log.InfoLevel )
    }

    hupData := coro_sighup()

    AddJSONLog( nil, config.Log.Main, 1, hupData )
    lineJsonLog := AddJSONLog( lineLogger1, config.Log.ProcLines, 2, hupData )
    lineTracker := lineJsonLog.inMemTracker
    
    return lineLogger, lineTracker
}

func coro_sighup() ( *HupData ) {
    hupData := HupData{
        hupA: false,
        hupB: false,
    }
    c := make(chan os.Signal, 2)
    signal.Notify(c, syscall.SIGHUP)
    go func() {
        for {
            <- c
            log.WithFields( log.Fields{
                "type": "sighup",
                "state": "begun",
            } ).Info("HUP requested")
            hupData.mutex.Lock()
            hupData.hupA = true
            hupData.hupB = true
            hupData.mutex.Unlock()
        }
    }()
    return &hupData
}