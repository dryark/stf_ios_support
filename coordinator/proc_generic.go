package main

import (
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
    curIP     string
    noRestart bool
    noWait    bool
    onStop    func( *RunningDev )
}

type GPMsg struct {
    msgType int
}

type GenericProc struct {
    controlCh chan GPMsg
    backoff *Backoff
    pid int
    cmd *gocmd.Cmd
}

func (self *GenericProc) Kill() {
    if self.cmd == nil { return }
    self.controlCh <- GPMsg{ msgType: 1 }
}

func (self *GenericProc) Restart() {
    if self.cmd == nil { return }
    self.controlCh <- GPMsg{ msgType: 2 }
}

func restart_proc_generic( devd *RunningDev, name string ) {
    genProc := devd.process[ name ]
    genProc.Restart()
}

func proc_generic( opt ProcOptions ) ( *GenericProc ) {
    controlCh := make( chan GPMsg )
    proc := GenericProc {
        controlCh: controlCh,
    }
    
    devd := opt.devd
    
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
        devd.lock.Lock()
        devd.process[ opt.procName ] = &proc
        devd.lock.Unlock()
    } else {
        plog = log.WithFields( log.Fields{ "proc": opt.procName } )
        lineLog = opt.lineLog.WithFields( log.Fields{ "proc": opt.procName } )
        opt.baseProgs.lock.Lock()
        opt.baseProgs.process[ opt.procName ] = &proc
        opt.baseProgs.lock.Unlock()
    }
  
    backoff := Backoff{}
    proc.backoff = &backoff

    stop := false
    
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

        cmd := gocmd.NewCmdOptions( gocmd.Options{ Streaming: true }, opt.binary, opt.args... )
        proc.cmd = cmd
        
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

        backoff.markStart()
        
        statCh := cmd.Start()
        
        proc.pid = cmd.Status().PID
        
        plog.WithFields( log.Fields{
            "type": "proc_pid",
            "pid": proc.pid,
        } ).Debug("Process pid")
        
        /*if err != nil {
            plog.WithFields( log.Fields{
                "type": "proc_err",
                "error": err,
            } ).Error("Error starting - " + opt.procName)

            proc.proc = nil
        }*/
        
        outStream := cmd.Stdout
        errStream := cmd.Stderr
        
        runDone := false
        for {
            select {
                case <- statCh:
                    runDone = true
                case msg := <- controlCh:
                    plog.Debug("Got stop request on control channel")
                    if msg.msgType == 1 { // stop
                        stop = true
                        proc.cmd.Stop()
                    } else if msg.msgType == 2 { // restart
                        proc.cmd.Stop()
                    }
                case line := <- outStream:
                    doLog := true
                    if opt.stdoutHandler != nil { doLog = opt.stdoutHandler( line, plog ) }
                    if doLog { lineLog.WithFields( log.Fields{ "line": line } ).Info(""); }
                case line := <- errStream:
                    doLog := true
                    if opt.stderrHandler != nil { doLog = opt.stderrHandler( line, plog ) }
                    if doLog { lineLog.WithFields( log.Fields{ "line": line, "iserr": true } ).Info("") }
            }
            if runDone { break }
        }
        
        proc.cmd = nil
        
        backoff.markEnd()

        plog.WithFields( log.Fields{ "type": "proc_end" } ).Warn("Process end - "+ opt.procName)
        
        if opt.onStop != nil {
            opt.onStop( devd )
        }
        
        if opt.noRestart { 
            plog.Debug( "No restart requested" )
            break
        }
        
        if stop { break }
        
        if !opt.noWait {
            backoff.wait()
        } else {
            plog.Debug("No wait requested")
        }
    } }()
    
    return &proc
}