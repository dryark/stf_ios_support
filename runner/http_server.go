package main

import (
    "bytes"
    "fmt"
    "net/http"
    "text/template"
    "strconv"
    "strings"
    "time"
    uj "github.com/nanoscopic/ujsonin/mod"
)

type Info struct {
    proc *GenericProc
    passmap map[string] string
}

func coro_http_server( port int, proc *GenericProc, passmap map[string] string, secure bool, crt string, key string, runnerVersion VersionInfo, config *uj.JNode ) {
    var listen_addr = fmt.Sprintf( "0.0.0.0:%d", port )
    startServer( listen_addr, proc, passmap, secure, crt, key, runnerVersion, config )
}

func BasicAuth(handler http.HandlerFunc, passmap map[string] string ) http.HandlerFunc {
	realm := "Enter auth for coordinator runner admin"
	
    return func(w http.ResponseWriter, r *http.Request) {

        user, pass, ok := r.BasicAuth()

        if !ok || !check_pass( user, pass, passmap ) {
            w.Header().Set("WWW-Authenticate", `Basic realm="`+realm+`"`)
            w.WriteHeader(401)
            w.Write([]byte("Unauthorised.\n"))
            return
        }

        handler(w, r)
    }
}

func startServer( listen_addr string, proc *GenericProc, passmap map[string] string, secure bool, crt string, key string, runnerVersion VersionInfo, config *uj.JNode ) {
	info := Info{
		proc: proc,
		passmap: passmap,
	}
	
    fmt.Printf("HTTP server started")

    rootClosure := BasicAuth( func( w http.ResponseWriter, r *http.Request ) {
        handleRoot( w, r, info, runnerVersion )
    }, passmap );
    startClosure := BasicAuth( func( w http.ResponseWriter, r *http.Request ) {
        handleStart( w, r, info )
    }, passmap );
    stopClosure := BasicAuth( func( w http.ResponseWriter, r *http.Request ) {
        handleStop( w, r, info )
    }, passmap );
    restartClosure := BasicAuth( func( w http.ResponseWriter, r *http.Request ) {
        handleRestart( w, r, info )
    }, passmap );
    updateClosure := BasicAuth( func( w http.ResponseWriter, r *http.Request ) {
        handleUpdate( w, r, info, config )
    }, passmap );
    
    http.HandleFunc( "/", rootClosure )
    http.HandleFunc( "/start", startClosure )
    http.HandleFunc( "/stop", stopClosure )
    http.HandleFunc( "/restart", restartClosure )
    http.HandleFunc( "/update", updateClosure )
    
    var err error
    if secure {
    	err = http.ListenAndServeTLS( listen_addr, crt, key, nil )
    } else {
    	err = http.ListenAndServe( listen_addr, nil )
    }
    fmt.Printf("HTTP ListenAndServe Error %s\n", err)
}

func handleRoot( w http.ResponseWriter, r *http.Request, info Info, rv VersionInfo ) {
	var allVersions map[string] VersionInfo = make( map[string] VersionInfo )
	allVersions["Runner"] = rv
    loadVersionInfo( allVersions )
    
	versionText := ""
	for itemName,item := range allVersions {
        var str bytes.Buffer
        
        remote := item.GitRemote
        remote = strings.Replace( remote, "git@github.com:", "", 1 )
        remote = strings.Replace( remote, ".git", "", 1 )
        rawRemote := remote
        remote = "<a href='https://github.com/" + remote + "'>" + remote + "</a>"
        
        commit := item.GitCommit
        commit = "<a href='https://github.com/" + rawRemote + "/commit/" + commit + "'>" + commit + "</a>"
        
        time := unixToTimeObject( item.GitDate )
        timeStr := time.Format( "Mon, Jan 2 2006 3:04 PM MST" )
        
        versionTpl.Execute( &str, map[string] string {
            "GitCommit": commit,
            "GitDate": timeStr,
            "GitRemote": remote,
            "EasyVersion": item.EasyVersion,
            "Name": itemName,
        } )
        versionText += str.String() + "<br>"
    } 
        
    rootTpl.Execute( w, map[string] string{
		"pid": strconv.Itoa( info.proc.pid ),
		"timeUp": info.proc.backoff.timeUpText(),
		"Versions": versionText,
    } )
}

func unixToTimeObject( unix string ) ( time.Time ) {
    i, _ := strconv.ParseInt( unix, 10, 64)
    return time.Unix( i, 0 )
}

func handleStart( w http.ResponseWriter, r *http.Request, info Info ) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    info.proc.Start()
    fmt.Fprintf( w, "ok" )
}

func handleStop( w http.ResponseWriter, r *http.Request, info Info ) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    info.proc.Stop()
    fmt.Fprintf( w, "ok" )
}

func handleRestart( w http.ResponseWriter, r *http.Request, info Info ) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    info.proc.Restart()
    fmt.Fprintf( w, "ok" )
}

func handleUpdate( w http.ResponseWriter, r *http.Request, info Info, config *uj.JNode ) {
    w.Header().Set("Content-Type", "text/html; charset=utf-8")
    runUpdate( info, w, config )
}

var versionTpl = template.Must(template.New("version").Parse(`
	{{.Name}} Version info:<br>
	<table cellpadding=3 cellspacing=1 border=1>
		<tr>
			<td>Git Commit</td>
			<td>{{.GitCommit}}</td>
		</tr>
		<tr>
			<td>Git Date</td>
			<td>{{.GitDate}}</td>
		</tr>
		<tr>
			<td>Git Remote</td>
			<td>{{.GitRemote}}</td>
		</tr>
		<tr>
			<td>Easy Version</td>
			<td>{{.EasyVersion}}</td>
		</tr>
	</table>
`))

var rootTpl = template.Must(template.New("root").Parse(`
<!DOCTYPE html>
<html>
	<head>
	  <script>
	  // following functions adapted from minlib ( https://github.com/nanoscopic/minlib )
	  function getel( id ) {
        return document.getElementById( id );
      }
      function newel( typ ) {
        return document.createElement( typ );
      }
      function append( a, b ) {
        a.appendChild( b );
      }
      function clear(e) {
        if(!e) return;
        while(e.firstChild) e.removeChild(e.firstChild);
      }
      function req( type, url, handler, body ) {
        var xhr = new XMLHttpRequest();
        xhr.open( type, url );
        xhr.responseType = 'json';
        xhr.onload = function(x) { handler(x,xhr); }
        if( type == 'POST' && body ) xhr.send(body);
        else xhr.send();
      }
      function c_restart() {
        req( 'POST', '/restart', function() {}, JSON.stringify( {} ) );
      }
      function c_start() {
        req( 'POST', '/start', function() {}, JSON.stringify( {} ) );
      }
      function c_stop() {
        req( 'POST', '/stop', function() {}, JSON.stringify( {} ) );
      }
      function c_update() {
        getel('updatebox').style.display = 'block';
        getel('frame').src = '/update';
      }
      function c_hide_update() {
        getel('updatebox').style.display = 'none';
      }
      function line( html ) {
        if( html.match(/^\s*$/) ) return;
        var out = getel('out');
        var span = newel('span');
        span.innerHTML = html + "<br>";
        append( out, span );
        scrollToBottom( out );
      }
      function text( html ) {
        if( html.match(/^\s*$/) ) return;
        var out = getel('out');
        var span = newel('span');
        span.innerHTML = html;
        append( out, span );
        scrollToBottom( out );
      }
      function scrollToBottom(el) {
        el.scrollTop = el.scrollHeight - el.clientHeight;
      }
      function updateStart() {
        clear( getel('out') );
      }
	  </script>
	</head>
	<body>
		<h2>DeviceFarmer IOS Coordinator Runner</h2>
		<div id='updatebox' style='display:none'>
            <iframe id='frame' style="width: 100%; height: 5px; overflow-y: hidden; display: none">
            </iframe><br>
            <div id='out' style="width: 100%; height: 300px; overflow-y: scroll">
            </div>
            <button onclick="c_hide_update()">Close Update Panel</button>
            <br><br>
        </div>
		PID: {{.pid}}<br>
		Time up: {{.timeUp}}<br>
		<button onclick="c_start()">Start</button><br>
		<button onclick="c_stop()">Stop</button><br>
		<button onclick="c_restart()">Restart</button><br>
		<button onclick="c_update()">Update</button><br>
		<hr>
		{{.Versions}}
	</body>
</html>
`))
