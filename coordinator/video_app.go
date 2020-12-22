package main

import (
    "fmt"
    "io/ioutil"
    "os"
    "os/exec"
)

func va_write_config( config *Config, uuid string, vidport string, ip string ) {
    // create a temp file containing config
    // use ios-deploy to write the file to Documents dir of app
    fmt.Printf("Writing video app config port=%s ip=%s\n", vidport, ip )
    
    conf := fmt.Sprintf(`{
    "port": "%s",
    "ip": "%s"
}`, vidport, ip )
    fh, err := ioutil.TempFile("", "config")
	if err != nil {
		os.Exit(1)
	}
	fh.WriteString( conf )
	//defer os.Remove( fh.Name() )
	
	fmt.Printf( "%s -i %s -o %s -1 %s -2 %s\n", config.BinPaths.IosDeploy, uuid, fh.Name(), "com.dryark.vidtest2", "Documents/config.json" );
		
	exec.Command( 
	    config.BinPaths.IosDeploy,
	    "-i", uuid,
	    "-o", fh.Name(),
	    "-1", "com.dryark.vidtest2",
	    "-2", "Documents/config.json",
	).Output()
	
}

func va_start_stream() {
}

func va_stop_stream() {
}

func va_check_status() {
}