package main

import (
	"crypto/tls"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"strings"
)

var getRemoteConn func() (net.Conn, error)

var listenAddr = flag.String(`listen`, `:8000`, `Listen address. Eg: :8000; unix:/tmp/tcp-tls.sock`)
var remoteAddr = flag.String(`remote`, `127.0.0.1:443`, `Remote address. Eg: 127.0.0.1:443; unix:/tmp/tls.sock`)
var skipVerify = flag.Bool("tls-skip-verify", true, "Skip verify TLS Server")

func proxyConn(localConn net.Conn) {
	remoteConn, err := getRemoteConn()
	if err != nil {
		log.Println(err)
		return
	}
	defer remoteConn.Close()
	defer localConn.Close()
	go io.Copy(localConn, remoteConn)
	io.Copy(remoteConn, localConn)
}

func main() {

	flag.Parse()

	var err error
	var ln net.Listener

	if strings.HasPrefix(*listenAddr, `unix:`) {
		unixFile := (*listenAddr)[5:]
		os.Remove(unixFile)
		ln, err = net.Listen(`unix`, unixFile)
		os.Chmod(unixFile, os.ModePerm)
		log.Println(`Listening:`, unixFile)
	} else {
		ln, err = net.Listen(`tcp`, *listenAddr)
		log.Println(`Listening:`, ln.Addr().String())
	}
	if err != nil {
		log.Panicln(err)
	}

	if *skipVerify {
		log.Println(`Skip verify TLS Server, to disable, use: -tls-skip-verify=0`)
	}
	tlsConfig := &tls.Config{
		InsecureSkipVerify: *skipVerify,
	}
	if strings.HasPrefix(*remoteAddr, `unix:`) {
		unixFile := (*remoteAddr)[5:]
		getRemoteConn = func() (net.Conn, error) {
			return tls.Dial("unix", unixFile, tlsConfig)
		}
		log.Println(`Proxying to`, unixFile)
	} else {
		getRemoteConn = func() (net.Conn, error) {
			return tls.Dial("tcp", *remoteAddr, tlsConfig)
		}
		log.Println(`Proxying to`, *remoteAddr)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		go proxyConn(conn)
	}
}
