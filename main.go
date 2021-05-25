package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
)

func main() {
	clientListener, remote, addrListener := parseFlags()

	// Client Listener
	ln, err := net.Listen("tcp", clientListener.String())
	if err != nil {
		log.Fatalf("could not open a TCP connection: %s", err)
	}
	log.Printf("== LISTENER STARTED ON %s:%d ==\n", clientListener.IP, clientListener.Port)

	// Address Updater Listener
	addrLn, err := net.Listen("tcp", addrListener.String())
	if err != nil {
		log.Fatalf("could not open a TCP connection: %s", err)
	}
	log.Printf("== ADDR LISTENER STARTED ON %s:%d ==\n", addrListener.IP, addrListener.Port)

	connDone := make(chan net.Addr)

	// Keep track of whether there's a client connected already.
	//
	// It shouldn't happen that this variable is modified concurrently at
	// any point in time.
	var ongoing bool

	// Waits for connection disconnect, then marks the connection free for
	// other connection to connect.
	go func() {
		for addr := range connDone {
			log.Println("CONNECTION ENDED WITH", addr)
			ongoing = false
		}
	}()

	// Listener goroutine for endpoint IP address update.
	go func() {
		var conn net.Conn
		for {
			// close the previous connection
			if conn != nil {
				conn.Close()
			}

			// continue listening for the next one
			conn, err = addrLn.Accept()
			if err != nil {
				log.Printf("a connection with address updater endpoint was attempted, but failed: %s\n", err)
			}

			data, err := io.ReadAll(conn)
			if err != nil {
				log.Printf("error reading from %s connected on the address updater endpoint: %s\n", conn.RemoteAddr(), err)
			}

			parsedIP := net.ParseIP(strings.Trim(string(data), "\n"))
			if parsedIP == nil {
				log.Printf("ip received on the updater endpoint is invalid (%s)\n", string(data))
				continue
			}

			oldIP := remote.IP
			remote.IP = parsedIP
			log.Printf("[!] REMOTE IP UPDATE %s => %s: ", oldIP, remote.IP)
		}
	}()

	// Listener for incoming clients. It will pass the connection to a
	// separate goroutine if the connection is eligible. If another
	// connection exists, it will end it immediately.
	for {
		conn, err := ln.(*net.TCPListener).Accept()
		if err != nil {
			log.Fatal(err)
		}

		if ongoing {
			log.Printf("ended connection with %s since the connection limit has been reached\n", conn.RemoteAddr())
			conn.Close()
		}

		log.Printf("CONNECTION ESTABLISHED WITH %s\n", conn.RemoteAddr())
		ongoing = true
		go handleConn(conn, *remote, connDone)
	}
}

// Parse flags and return 3 addresses:
// - address which will be used for the listener for the client connection
// - address of the remote server which this app is proxying
// - address which will be used for the listener for the endpoint which will receive remote server address updates
func parseFlags() (*net.TCPAddr, *net.TCPAddr, *net.TCPAddr) {
	clientListerSockF := flag.String("clientListener", ":3110", "socket on which this machine listens for incoming connections, format address:port. The default is :3009")
	remoteSockF := flag.String("initialRemoteAddr", ":22", "initial remote address of the remote server. The default is :22")
	addrUpdateSockF := flag.String("addrUpdateListener", ":3111", "socket which listens for the updates of the IP that this machine is proxying. The default is :3010")

	flag.Parse()

	clientListerSock, err := parseSocket(*clientListerSockF)
	if err != nil {
		log.Fatalf("error parsing the listener socket: %s", err)
	}
	remoteSock, err := parseSocket(*remoteSockF)
	if err != nil {
		log.Fatalf("error parsing the remote socket: %s", err)
	}
	//addrUpdateSock, err := getLocalIP()
	addrUpdateSock, err := parseSocket(*addrUpdateSockF)
	if err != nil {
		log.Fatalf("could not get local IP: %s", err)
	}

	return clientListerSock, remoteSock, addrUpdateSock
}

// Handle the connection requested by client.
func handleConn(conn net.Conn, socket net.TCPAddr, done chan<- net.Addr) {
	// Clean up the connection when either side is closed
	defer func() {
		conn.Close()
		if done != nil {
			done <- conn.RemoteAddr()
		}
	}()

	syncChan := make(chan interface{})

	var outbound net.Conn

	// Client-server communication
	go func() {
		var err error
		outbound, err = net.Dial("tcp", socket.String())
		if err != nil {
			log.Fatalf("outbound connection refused: %s", outbound)
		}
		defer outbound.Close()

		syncChan <- 1

		reader := bufio.NewReader(outbound)
		reader.WriteTo(conn)

		syncChan <- 1
	}()

	<-syncChan

	// Server-client communication
	reader := bufio.NewReader(conn)
	reader.WriteTo(outbound)

	<-syncChan
}

// Get the IP of the active net interface.
//func getLocalIP() (net.IP, error) {
//	ifaces, err := net.Interfaces()
//	if err != nil {
//		return nil, err
//	}
//	for i := range ifaces {
//		iface := ifaces[len(ifaces)-i-1]
//		addrs, err := iface.Addrs()
//		if err != nil {
//			return nil, err
//		}
//		for _, addr := range addrs {
//			switch v := addr.(type) {
//			case *net.IPNet:
//				return v.IP, nil
//			case *net.IPAddr:
//				return v.IP, nil
//			}
//		}
//	}
//	return nil, errors.New("could not read IP from either interface")
//}

// Check is the ip:port configuration valid. It will return an error
// for any address that contains a colon (IPv6, MAC)
func parseSocket(sock string) (*net.TCPAddr, error) {
	if sock == "" {
		return nil, errors.New("remote socket not passed")
	}
	parts := strings.Split(sock, ":")
	if len(parts) != 2 {
		return nil, fmt.Errorf("expected 2 parts (ip and port), got %d", len(parts))
	}

	var ip net.IP = []byte{0, 0, 0, 0} // empty means that the address is 0.0.0.0
	if parts[0] != "" {
		ip = net.ParseIP(parts[0])
		if ip == nil {
			return nil, fmt.Errorf("couldn't parse the IP (%s)", parts[0])
		}
	}

	portInt, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("port %s isn't a number", parts[1])
	}

	return &net.TCPAddr{IP: ip, Port: portInt, Zone: ""}, nil

}
