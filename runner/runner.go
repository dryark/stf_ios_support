package main

import (
    "fmt"
    "time"
    "flag"
    gocmd "github.com/go-cmd/cmd"
    uj "github.com/nanoscopic/ujsonin/mod"
    "io/ioutil"
    "os"
    "os/exec"
)

type GPMsg struct {
    msgType int
}

type GenericProc struct {
    controlCh chan GPMsg
    backoff *Backoff
    pid int
    cmd *gocmd.Cmd
    hold bool
}

func (self *GenericProc) Kill() {
    if self.cmd == nil { return }
    self.controlCh <- GPMsg{ msgType: 1 }
}

func (self *GenericProc) Restart() {
    if self.cmd == nil { return }
    self.controlCh <- GPMsg{ msgType: 2 }
}

func (self *GenericProc) Start() {
    self.controlCh <- GPMsg{ msgType: 3 }
}

func (self *GenericProc) Stop() {
    self.controlCh <- GPMsg{ msgType: 4 }
}

func proc_generic( binary string, args []string, startDir string ) ( *GenericProc ) {
    controlCh := make( chan GPMsg )
    backoff := Backoff{}
    
    proc := GenericProc {
        controlCh: controlCh,
        backoff: &backoff,
        hold: false,
    }
    
    stop := false
    hold := false
    
    go func() { for {
        if hold == true {
            fmt.Println("Waiting for signal to start again")
        }
        
        for {
            if hold == false {
                break
            }
            select {
                case msg := <- controlCh:
                    fmt.Printf("Got message on control channel")
                    if msg.msgType == 3 { // start
                        hold = false
                        break
                    }
            }
        }
        
        fmt.Printf("Coordinator start\n")
        
        if !fileExists( binary ) {
            fmt.Printf("Coordinator binary does not exist. Waiting for creation\n")
            hold = true
            continue
        }
        cmd := gocmd.NewCmdOptions( gocmd.Options{ Streaming: true }, binary, args... )
        proc.cmd = cmd
        
        if startDir != "" {
            cmd.Dir = startDir
        }
        
        backoff.markStart()
        
        statCh := cmd.Start()
        
        i := 0
        for {
            proc.pid = cmd.Status().PID
            if proc.pid != 0 {
                break
            }
            time.Sleep(50 * time.Millisecond)
            if i > 4 {
                break
            }
        }
        
        fmt.Printf("PID %d\n", proc.pid)
                
        outStream := cmd.Stdout
        errStream := cmd.Stderr
        
        runDone := false
        for {
            select {
                case <- statCh:
                    runDone = true
                case msg := <- controlCh:
                    fmt.Printf("Got stop request on control channel\n")
                    typ := msg.msgType
                    
                    if typ == 1 { // stop
                        stop = true
                    } else if typ == 4 { // stop
                        hold = true
                    }
                    
                    if typ == 1 || typ == 2 || typ == 4 {
                        proc.cmd.Stop()
                        cleanup_subprocs( binary )
                    }
                    
                case line := <- outStream:
                    fmt.Println( line )
                case line := <- errStream:
                    fmt.Println( line )
            }
            if runDone { break }
        }
        
        proc.cmd = nil
        proc.pid = 0
        
        backoff.markEnd()

        fmt.Printf("Coordinator end\n")
                
        if stop { break }
        backoff.wait()
    } }()
    
    return &proc
}

func cleanup_subprocs( binary string ) {
    // Make absolutely sure all coordinator subprocesses have been stopped
    out, _ := exec.Command( binary, "-killProcs" ).Output()
    fmt.Println( out )
}

func gen_cert() {
    out, err := exec.Command( "/usr/bin/perl", "gencert.pl" ).Output()
    if err != nil {
        fmt.Printf("Error from cert gen: %s\n", err )
        return
    }
    fmt.Println( out )
}

var GitCommit string
var GitDate string
var GitRemote string
var EasyVersion string

type VersionInfo struct {
    GitCommit string
    GitDate string
    GitRemote string
    EasyVersion string
}

func main() {
    runnerVersion := VersionInfo{
        GitCommit: GitCommit,
        GitDate: GitDate,
        GitRemote: GitRemote,
        EasyVersion: EasyVersion,
    }
    
	var passToHash = flag.String( "pass", "", "Password to show hash of" )
	var doVersion  = flag.Bool( "version"   , false        , "Show coordinator version info" )
	
	flag.Parse()
	
	if *passToHash != "" {
	    hash := hash_pass( *passToHash )
	    fmt.Printf("hash:%s\n", hash )
	    return
	}
	if *doVersion {
        fmt.Printf("Commit:%s\nDate:%s\nRemote:%s\nVersion:%s\n", GitCommit, GitDate, GitRemote, EasyVersion )
        os.Exit(0)
    }
    
    if !fileExists("server.crt") {
        gen_cert()
    }
	
    content, _ := ioutil.ReadFile("runner.json")
    root, _ := uj.Parse( content )
    users := root.Get("users")
    passmap := json_users_to_passmap( users )
    secure := root.Get("https").Bool()
    installDir := root.Get("install_dir").String()
    
    coordPath := installDir + "/bin/coordinator"
    cleanup_subprocs( coordPath )
    cleanup_procs()
	
	proc := proc_generic( coordPath, []string{}, installDir )
	
	coro_sigterm( proc, coordPath )
	
	if !fileExists( "runner.json" ) {
	    fmt.Println("runner.json config file not present. exiting\n")
	    os.Exit( 1 )
	    return
	}
	
    crt := ""
    key := ""
    if secure {
        crt = root.Get("crt").String()
        key = root.Get("key").String()
    }
    
	coro_http_server( 8021, proc, passmap, secure, crt, key, runnerVersion, root )
}