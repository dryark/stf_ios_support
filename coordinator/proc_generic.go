package main

import (
  "os/exec"
  log "github.com/sirupsen/logrus"
  gocmd "github.com/go-cmd/cmd"
)

type OutputHandler func( string, *log.Entry ) (bool)

type ProcOptions struct {
    config    *Config
    baseProgs *BaseProgs
    devd      *RunningDev
    lineLog   *log.Entry
    procName  string
    binary    string
    args      []string
    stderrHandler OutputHandler
    stdoutHandler OutputHandler
    startFields log.Fields
    startDir  string
    env       map[string]string
}

func proc_generic( opt ProcOptions ) {
    devd := opt.devd.dup()
    var plog *log.Entry
    var lineLog *log.Entry
    if devd != nil {
        plog = log.WithFields( log.Fields{
            "proc": opt.procName,
            "uuid": censor_uuid( devd.uuid ),
        } )
        lineLog = opt.lineLog.WithFields( log.Fields{
            "proc": opt.procName,
            "uuid": censor_uuid( devd.uuid ),
        } )
    } else {
        plog = log.WithFields( log.Fields{ "proc": opt.procName } )
        lineLog = opt.lineLog.WithFields( log.Fields{ "proc": opt.procName } )
    }
    
    shuttingDown := opt.devd.getShuttingDown( opt.baseProgs )
    if shuttingDown || gStop { return }
    
    backoff := Backoff{}
    opt.devd.setBackoff( opt.procName, &backoff, opt.baseProgs )
    
    stdoutChan := make(chan string, 100)
    stderrChan := make(chan string, 100)

    stop := false
    
    go func() { for {
        line := <- stderrChan
        doLog := true
        if opt.stderrHandler != nil { doLog = opt.stderrHandler( line, plog ) }
        if doLog { lineLog.WithFields( log.Fields{ "line": line, "iserr": true } ).Info("") }
        if stop { break }
    } } ()
    
    go func() { for {
        line := <- stdoutChan
        doLog := true
        if opt.stdoutHandler != nil { doLog = opt.stdoutHandler( line, plog ) }
        if doLog { lineLog.WithFields( log.Fields{ "line": line } ).Info(""); }
        if stop { break }
    } }()
    
    stdStream := gocmd.NewOutputStream(stdoutChan)
    errStream := gocmd.NewOutputStream(stderrChan) 
    
    go func() { for {
        startFields := log.Fields{
            "type":     "proc_start",
            "binary":   opt.binary,
        }
        if opt.startFields != nil {
            for k, v := range opt.startFields {
                startFields[k] = v
            }
        }
        
        plog.WithFields( startFields ).Info("Process start - " + opt.procName)

        cmd := exec.Command( opt.binary, opt.args... )
        
        if opt.startDir != "" {
            cmd.Dir = opt.startDir
        }
        
        if opt.env != nil {
            var envArr []string
            for k,v := range( opt.env ) {
                envArr = append( envArr, k, v )
            }
            cmd.Env = envArr
        }

        cmd.Stdout = stdStream
        cmd.Stderr = errStream

        backoff.markStart()
        
        err := cmd.Start()
        if err != nil {
            plog.WithFields( log.Fields{
                "type": "proc_err",
                "error": err,
            } ).Error("Error starting - " + opt.procName)

            opt.devd.setProcess( opt.procName, nil, opt.baseProgs)
        } else {
            opt.devd.setProcess( opt.procName, cmd.Process, opt.baseProgs )
        }
        
        cmd.Wait()
        backoff.markEnd()

        plog.WithFields( log.Fields{  "type": "proc_end" } ).Warn("Process end - "+ opt.procName)
        
        opt.devd.setProcess( opt.procName, nil, opt.baseProgs )
        shuttingDown := opt.devd.getShuttingDown( opt.baseProgs )
        
        if shuttingDown {
            stop = true
            break
        }
        
        backoff.wait()
    } }()
}