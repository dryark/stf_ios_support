package main

import (
	"net/http"
	"fmt"
)

func main() {
	fs := http.FileServer( http.Dir( "./updates" ) )
	fmt.Println( http.ListenAndServe( ":8022", fs ) )
}