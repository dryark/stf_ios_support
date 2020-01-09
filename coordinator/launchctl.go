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
)

type Launcher struct {
    label string
    arguments []string
    keepalive bool
    stdout string
    stderr string
    cwd string
    file string
    lock sync.Mutex
}

func NewLauncher( label string, arguments []string, keepalive bool, cwd string ) (*Launcher) {
    file := label // strings.ReplaceAll( label, ".", "_" )
    user, _ := user.Current()
    file = fmt.Sprintf("%s/Library/LaunchAgents/%s.plist", user.HomeDir, file)
    //strings.Replace
    launcher := Launcher{
        label: label,
        arguments: arguments,
        keepalive: keepalive,
        stdout: "/dev/null",
        stderr: "/dev/null",
        cwd: cwd,
        file: file,
    }
    return &launcher    
}

func ( self *Launcher ) pid() (pid int) {
    output, _ := exec.Command(fmt.Sprintf("launchctl list %s", self.label)).Output()
    lines := strings.Split( string(output), "\n" )
    
    pid = 0
    for _, line := range lines {
        if strings.Contains( line, "\"PID\"" ) {
            pos := strings.Index( line, "\"PID\"" )
            pos = pos + 7
            
            val := line[pos:len(line)-2]
            pid, _ = strconv.Atoi( val )
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
    
    var data bytes.Buffer
    launchTpl.Execute( &data, map[string] string {
        "label":  self.label,
        "arguments": argx,
        "keepalive": keepaliveX,
        "stdout": self.stdout,
        "stderr": self.stderr,
        "cwd": self.cwd,
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
  
  <key>LimitLoadToSessionType</key>
  <string>Aqua</string>
</dict>
</plist>
`))