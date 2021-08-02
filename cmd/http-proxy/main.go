package main

import (
	"flag"
	"log"
	"net"
	"net/http"

	"github.com/spinzed/proxytools/internal"
)

type Proxy struct{}

func (p *Proxy) ServeHTTP(wr http.ResponseWriter, req *http.Request) {
	log.Println(req.RemoteAddr, "", req.Method, "", req.URL)

	if req.Method == "CONNECT" {
		handleHTTPS(wr, req)
    } else {
        handleHTTP(wr, req)
    }
}

// Setup two http connections, bodies are copied from one to another.
// The connection is not end-to-end encrypted.
func handleHTTP(wr http.ResponseWriter, req *http.Request) {
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		msg := "unsupported protocol scheme " + req.URL.Scheme
		http.Error(wr, msg, http.StatusBadRequest)
		log.Println(msg)
		return
	}

    // Client which will connect to the actual destination.
    // The request that will be used with this client is the same one
    // that is used for the client-proxy connection with some modified values.
	client := &http.Client{}

	//http: Request.RequestURI can't be set in client requests.
	//http://golang.org/src/pkg/net/http/client.go
	req.RequestURI = ""

    // Delete hop by hop headers
	internal.DelHopHeaders(req.Header)

    // If there wasn't any error getting the remote address, append this
    // proxy to the x-forwarded-for proxy IP list
	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		internal.AppendHostToXForwardHeader(req.Header, clientIP)
	}

    // Make the request with the original request object
	resp, err := client.Do(req)
	if err != nil {
		http.Error(wr, "Server Error", http.StatusInternalServerError)
		log.Println("ServeHTTP error: ", err)
		return
	}
	defer resp.Body.Close()

	log.Println(req.RemoteAddr, " ", resp.Status)

    // Remove hop by hop headers
	internal.DelHopHeaders(resp.Header)

    // Copy the rest of the headers to the response
	internal.CopyHeader(wr.Header(), resp.Header)

    // Pass the proxy-server response status through the client-proxy connection.
    // When the client recieves this, it will start sending the information that
    // needs to be carried over to the client/next hop.
	wr.WriteHeader(resp.StatusCode)

    // Copy the data from server-proxy response and send it to the client
	internal.CopyData(resp.Body, wr)
}

// Setup a layer 4 proxy tunnel for https connections. The proxy cannot
// read the content, the conversation is end-to-end encrypted.
func handleHTTPS(wr http.ResponseWriter, req *http.Request) {
	serverConn, err := internal.MakeTCPConn(req.URL.Host)
	if err != nil {
		wr.WriteHeader(http.StatusServiceUnavailable)
		log.Println("Error trying to make a connection with the server:", err)
		return
	}
	wr.WriteHeader(http.StatusOK)

    // Check if it's possible to extract the underlying TCP connection object.
    // It should always be possible so this shouldn't fail.
	hijacker, ok := wr.(http.Hijacker)
	if !ok {
		http.Error(wr, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

    // Extract the underlying connection object
	clientConn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(wr, err.Error(), http.StatusServiceUnavailable)
	}

    // Setup routines which will copy recieved data from one connection to
    // another, one routine for each direction. After the connection end,
    // they will be closed.
	go internal.CopyAndClose(clientConn, serverConn)
	go internal.CopyAndClose(serverConn, clientConn)
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
