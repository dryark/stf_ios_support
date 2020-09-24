package main

import (
	"net/http"
	//"time"
	"fmt"
	"io"
	"os"
	uj "github.com/nanoscopic/ujsonin/mod"
	"io/ioutil"
	"strings"
	gocmd "github.com/go-cmd/cmd"
	escape "github.com/gorilla/template/v0/escape"
)

func writeLine( w http.ResponseWriter, f http.Flusher, str string, args ...interface{} ) {
    fmt.Fprintf( w, "<script>line(\"" )
    str = fmt.Sprintf( str, args... ) 
    escape.JSEscape( w, []byte(str) )
    fmt.Fprintf( w, "\")</script>" )
    f.Flush()
}

func writeText( w http.ResponseWriter, f http.Flusher, str string, args ...interface{} ) {
    fmt.Fprintf( w, "<script>text(\"" )
    str = fmt.Sprintf( str, args... ) 
    escape.JSEscape( w, []byte(str) )
    fmt.Fprintf( w, "\")</script>" )
    f.Flush()
}

func runUpdate( info Info, w http.ResponseWriter, config *uj.JNode ) {
	fw, ok := w.(http.Flusher)
	if !ok {
		fmt.Fprintf( w, "sadness. broken. :(" )
		return
	}
	
	//"updates": "/Users/user/stf_updates",
    //"install_dir": "/Users/user/stf",
    //"config": "/Users/user/stf/config.json"
	
    fmt.Fprintf( w, "<script>function line(t) { parent.line(t) };function text(t) { parent.text(t) } parent.updateStart()</script>" )
    
    writeLine( w, fw, "Stopping coordinator" )
    info.proc.Stop()
    
    installDir := config.Get("install_dir").String()
    configFile := config.Get("config").String()
    updateHost := config.Get("update_host").String()
    updatePort := config.Get("update_port").String()
    updateUrl := "http://" + updateHost + ":" + updatePort + "/"
    
    configWithin := false
    if strings.HasPrefix( configFile, installDir ) {
        configWithin = true
    }
    
    configSource := configFile
    
    lineSpace := "<div style='display:inline-block; width: 50px;'>&nbsp;</div>"
    
    if dirExists( installDir ) {
        writeLine( w, fw, "Install directory exists; erasing %s", installDir )
        if configWithin {
            writeLine( w, fw, lineSpace + "Config file within install dir; backing up %s", configFile )
            tempFile, err := ioutil.TempFile( "/tmp", "config_json" )
            if err != nil {
                writeLine( w, fw, err.Error() )
            }
            err = copyFileContents( configFile, tempFile.Name() )
            if err != nil {
                writeLine( w, fw, err.Error() )
            }
            configSource = tempFile.Name()
        }
        os.RemoveAll( installDir )
    }
    
    writeLine( w, fw, "Creating install directory %s", installDir )
    os.MkdirAll( installDir, 0755 )
    
    //writeLine( w, fw, "Downloading update information" )
    
    updateFolder := config.Get("updates").String()
    if !dirExists( updateFolder ) {
        writeLine( w, fw, "Update folder %s did not exist. Created", updateFolder )
        os.MkdirAll( updateFolder, 0755 )
    }
    
    updatesDest := updateFolder + "/updates.json"
    updatesSrc := updateUrl + "updates.json"
    err := download( updatesDest, updatesSrc )
    if err != nil {
        writeLine( w, fw, "Error downloading update metadata from %s: %s\n", updatesSrc, err )
        return
    }
    updateContent, _ := ioutil.ReadFile( updatesDest )
    uRoot, _ := uj.Parse( updateContent )
    latest := uRoot.Get("latest").String()
    writeLine( w, fw, "Latest update:%s", latest )
    
    latestDest := updateFolder + "/" + latest
    download( latestDest, updateUrl + latest )
    latestContent, _ := ioutil.ReadFile( latestDest )
    lRoot, _ := uj.Parse( latestContent )
    files := lRoot.Get("files")
    //writeLine( w, fw, "Files in update:" )
    fileArr := []string{}
    files.ForEach( func( fileNode *uj.JNode ) {
        file := fileNode.String()
        //writeLine( w, fw, lineSpace + file )
        fileArr = append( fileArr, file )
    } )
    
    writeLine( w, fw, "Downloading files:" )
    for _,file := range fileArr {
        dest := updateFolder + "/" + file
        writeText( w, fw, lineSpace + file + "..." )
        src := updateUrl + file
        download( dest, src )
        writeLine( w, fw, " Done" )
    }
    
    action := lRoot.Get("action").String()
    writeLine( w, fw, "Running install ( %s )", action )
    parts := strings.Split( action, " " )
    parts[0] = updateFolder + "/" + parts[0]
    
    os.Chmod( parts[0], 0770 )
    cmd := gocmd.NewCmdOptions( gocmd.Options{ Streaming: true }, parts[0], parts[1:]... )
    
    env := map[string] string {
        "CONFIG_SRC": configSource,
        "INSTALL_DIR": installDir,
        "UPDATE_DIR": updateFolder,
    }
    
    var envArr []string
    for k,v := range( env ) {
        envArr = append( envArr, k + "=" + v )
    }
    cmd.Env = envArr
    
    statCh := cmd.Start()
    
    outStream := cmd.Stdout
    errStream := cmd.Stderr
    runDone := false
    for {
        select {
            case <- statCh:
                runDone = true
            case line := <- outStream:
                line = strings.Replace( line, "[32m", "<font color='green'>", 1 )
                line = strings.Replace( line, "[91m", "<font color='red'>", 1 )
                line = strings.Replace( line, "[0m", "</font>", 1 )
                writeLine( w, fw, line )
            case line := <- errStream:
                writeLine( w, fw, "err:%s", line )
        }
        if runDone { break }
    }
    
    writeLine( w, fw, "Install complete" )
    
    if configWithin {
        os.Remove( configSource )
    }
    
    writeLine( w, fw, "Starting coordinator" )
    info.proc.Start()
}

// copyFileContents copies the contents of the file named src to the file named
// by dst. The file will be created if it does not already exist. If the
// destination file exists, all it's contents will be replaced by the contents
// of the source file.
func copyFileContents(src, dst string) (err error) {
    in, err := os.Open(src)
    if err != nil {
        return
    }
    defer in.Close()
    out, err := os.Create(dst)
    if err != nil {
        return
    }
    defer func() {
        cerr := out.Close()
        if err == nil {
            err = cerr
        }
    }()
    if _, err = io.Copy(out, in); err != nil {
        return
    }
    err = out.Sync()
    return err
}

func download( dest string, url string) error {
	resp, err := http.Get( url )
	if err != nil { return err }
	defer resp.Body.Close()

	out, err := os.Create( dest )
	if err != nil {	return err }
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}