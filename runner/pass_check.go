package main

import (
	"fmt"
	"crypto/sha256"
	"crypto/subtle"
	uj "github.com/nanoscopic/ujsonin/mod"
)

func check_pass( user string, pass string, hashes map[string] string ) bool {
	hash := hash_pass( pass )
	check, ok := hashes[ user ]
	if !ok { return false }
	res := subtle.ConstantTimeCompare([]byte(check),[]byte(hash))
	if res == 1 { return true }
	return false
}

func hash_pass( pass string ) string {
	h := sha256.New()
	h.Write([]byte(pass))
	hash := fmt.Sprintf("%x", h.Sum(nil))
	return hash
}

func json_users_to_passmap( users *uj.JNode ) ( map[string] string ) {
	passmap := map[string] string{}
	
	users.ForEach( func( cur *uj.JNode ) {
		//cur.Dump()
		user := cur.Get("user").String()
		pass := cur.Get("pass").String()
		passmap[ user ] = pass
	} )
	return passmap
}