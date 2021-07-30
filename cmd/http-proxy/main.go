package main

import (
	"flag"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
)

// Hop-by-hop headers. These are removed when sent to the backend.
// http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
var hopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te", // canonicalized version of "TE"
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func delHopHeaders(header http.Header) {
	for _, h := range hopHeaders {
		header.Del(h)
	}
}

func appendHostToXForwardHeader(header http.Header, host string) {
	// If we aren't the first proxy retain prior
	// X-Forwarded-For information as a comma+space
	// separated list and fold multiple headers into one.
	if prior, ok := header["X-Forwarded-For"]; ok {
		host = strings.Join(prior, ", ") + ", " + host
	}
	header.Set("X-Forwarded-For", host)
}

func copyData(source, dest net.Conn) {
    defer source.Close()
    defer dest.Close()
    io.Copy(dest, source)
}

type Proxy struct {}

func (p *Proxy) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	log.Println(req.RemoteAddr, "", req.Method, "", req.URL)

    // https request, set up layer 4 TCP tunnel
    if req.Method == "CONNECT" {
        setupTunnel(wr, req)
        return
    }

	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
	  	msg := "unsupported protocol scheme "+req.URL.Scheme
		http.Error(wr, msg, http.StatusBadRequest)
		log.Println(msg)
		return
	}

	client := &http.Client{}

	//http: Request.RequestURI can't be set in client requests.
	//http://golang.org/src/pkg/net/http/client.go
	req.RequestURI = ""

	delHopHeaders(req.Header)

	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		appendHostToXForwardHeader(req.Header, clientIP)
	}

	resp, err := client.Do(req)
	if err != nil {
		http.Error(wr, "Server Error", http.StatusInternalServerError)
		log.Println("ServeHTTP error: ", err)
        return
	}
	defer resp.Body.Close()

	log.Println(req.RemoteAddr, " ", resp.Status)

	delHopHeaders(resp.Header)

	copyHeader(wr.Header(), resp.Header)
	wr.WriteHeader(resp.StatusCode)
	io.Copy(wr, resp.Body)
}

func setupTunnel(wr http.ResponseWriter, req *http.Request) {
    serverConn, err := net.Dial("tcp", req.URL.Host)
    if err != nil {
        wr.WriteHeader(http.StatusServiceUnavailable)
        log.Println("Error trying to make a connection with the server:", err)
        return
    }
    wr.WriteHeader(http.StatusOK)

    hijacker, ok := wr.(http.Hijacker)
    if !ok {
        http.Error(wr, "Hijacking not supported", http.StatusInternalServerError)
        return
    }
    
    clientConn, _, err := hijacker.Hijack()
    if err != nil {
        http.Error(wr, err.Error(), http.StatusServiceUnavailable)
    }

    go copyData(clientConn, serverConn)
    go copyData(serverConn, clientConn)
    return
}

func main() {
	var addr = flag.String("addr", ":3128", "The addr of the application.")
	var https = flag.Bool("https", false, "Setup whether client needs to use https instead of http to connect to the proxy. If this is set, then path to pem and key must be set.")
	var pem = flag.String("pem", "", "Absolute path to the pem file.")
	var key = flag.String("key", "", "Absolute path to the key file.")
	flag.Parse()
	
    handler := &Proxy{}
	
	log.Println("Starting proxy server on", *addr)
    
    if *https {
        if err := http.ListenAndServeTLS(*addr, *pem, *key, handler); err != nil {
            log.Fatal("ListenAndServe:", err)
        }
    } else {
        if err := http.ListenAndServe(*addr, handler); err != nil {
            log.Fatal("ListenAndServe:", err)
        }
    }
}
