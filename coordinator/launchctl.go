package main

import (
  "bytes"
  "fmt"
  "io/ioutil"
  "os"
  "os/exec"
  "os/user"
  "strconv"
  "strings"
  "sync"
  "text/template"
  log "github.com/sirupsen/logrus"
  //ps "github.com/jviney/go-proc"
)

type Launcher struct {
    label string
    arguments []string
    keepalive bool
    stdout string
    stderr string
    cwd string
    file string
    asRoot bool
    lock sync.Mutex
}

func NewLauncher( label string, arguments []string, keepalive bool, cwd string, asRoot bool ) (*Launcher) {
    file := label // strings.ReplaceAll( label, ".", "_" )
    user, _ := user.Current()
    if asRoot == true {
        file = fmt.Sprintf("/Library/LaunchDaemons/%s.plist", file)
    } else {
        file = fmt.Sprintf("%s/Library/LaunchAgents/%s.plist", user.HomeDir, file)
    }
    //strings.Replace
    launcher := Launcher{
        label: label,
        arguments: arguments,
        keepalive: keepalive,
        stdout: "/dev/null",
        stderr: "/dev/null",
        cwd: cwd,
        file: file,
        asRoot: asRoot,
    }
    return &launcher    
}

func ( self *Launcher ) pid() (pid int) {
    user, _ := user.Current()
    pid = 0
    
    //log.WithFields( log.Fields{ "type": "blah", "asroot": self.asRoot, "user": user.Username } ).Info("fdfsdfds")
    
    if self.asRoot && user.Username != "root" {
        // trying to find information on a root owned plist, but not running as root
        // cannot use launchctl as a result :(
        
        fullCmdLine := strings.Join( self.arguments, " " )
        
        // This code doesn't work. Why? Who knows. Apparently go-proc can't retrieve processes run by root???
        /*procs := ps.GetAllProcessesInfo()
        for _, proc := range procs {
            testCmd := proc.Command + " " + strings.Join( proc.CommandLine, " " )
            //log.WithFields( log.Fields{ "type": "proc", "proc": testCmd } ).Info("proc")
            if proc.Pid == 15454 { //strings.Contains( testCmd, "openvpn" ) {
                log.WithFields( log.Fields{ "type": "testeq", "find": fullCmdLine, "have": testCmd } ).Info("testeq")
            }
            if testCmd == fullCmdLine {
                return proc.Pid
            }
        }*/
        cmd := exec.Command("/bin/ps","-Af")
        output, _ := cmd.Output()
        
        lines := strings.Split( string( output ), "\n" )
        for _, line := range lines {
            parts := strings.Split( line, " " )
            var nodup [] string
            for _, part := range parts {
                if part != "" {
                    nodup = append( nodup, part )
                }
            }
            line = strings.Join( nodup, " " )
            
            if strings.Contains( line, fullCmdLine ) {
                sp1 := strings.Index( line, " " )
                rest := line[ sp1: ]
                sp2 := strings.Index( rest, " " )
                rest = rest[ sp2 + 1 : ]
                sp3 := strings.Index( rest, " " )
                pid := rest[ 0: sp3 - 1 ]
                pidNum, _ := strconv.Atoi( pid )
                return pidNum
            }
        }
    } else {
        output, _ := exec.Command(fmt.Sprintf("launchctl list %s", self.label)).Output()
        lines := strings.Split( string(output), "\n" )
        
        for _, line := range lines {
            if strings.Contains( line, "\"PID\"" ) {
                pos := strings.Index( line, "\"PID\"" )
                pos = pos + 7
                
                val := line[pos:len(line)-2]
                pid, _ = strconv.Atoi( val )
            }
        }
    }
    return pid
}

func ( self *Launcher ) load() {
    // unload the service if it is already loaded
    pid := self.pid()
    if pid != 0 {
        self.unload()
    }
    
    argx := ""
    for _, arg := range self.arguments {
        argx += fmt.Sprintf("<string>%s</string>", arg)
    }
    
    keepaliveX := "<false/>"
    if self.keepalive {
        keepaliveX = "<true/>"
    }
    
    limitSession := ""
    if self.asRoot != true {
        limitSession = "<key>LimitLoadToSessionType</key>\n  <string>Aqua</string>\n"
    }
    
    var data bytes.Buffer
    launchTpl.Execute( &data, map[string] string {
        "label":  self.label,
        "arguments": argx,
        "keepalive": keepaliveX,
        "stdout": self.stdout,
        "stderr": self.stderr,
        "cwd": self.cwd,
        "limitSession": limitSession,
    } )
    
    // create / recreate the plist file
    err := ioutil.WriteFile( self.file, data.Bytes(), 0600 )
    if err != nil {
        log.WithFields( log.Fields{
            "type": "launch_err",
            "file": self.file,
            "error": err,
        } ).Error("Error writing plist file")
    }
    
    // load it
    self.lock.Lock()
    exec.Command("/bin/launchctl","load",self.file).Run()
    self.lock.Unlock()
}

func ( self *Launcher ) unload() {
    // unload
    self.lock.Lock()
    exec.Command("/bin/launchctl","unload",self.file).Run()
    
    // delete the file
    os.Remove(self.file)
    self.lock.Unlock()
}

var launchTpl = template.Must(template.New("launchfile").Parse(`
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>{{.label}}</string>
  
  <key>ProgramArguments</key>
  <array>
{{.arguments}}
  </array>
  
  <key>KeepAlive</key>
  {{.keepalive}}
  
  <key>StandardOutPath</key>
  <string>{{.stdout}}</string>
  
  <key>StandardErrorPath</key>
  <string>{{.stderr}}</string>
  
  <key>WorkingDirectory</key>
  <string>{{.cwd}}</string>
  
  {{.limitSession}}
</dict>
</plist>
`))