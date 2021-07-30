package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
)

// Addr is basically net.TCPAddr, but the IP isn't a byte slice, but a string.
// This allows IP to be an unresolved DNS entry.
type Addr struct {
    IP string
    Port int
}

func (a Addr) String() string {
    return a.IP + ":" + strconv.Itoa(a.Port)
}

func main() {
	clientListener, remote, addrListener, MAX_CONNS := parseFlags()

	// Client Listener
	ln, err := net.Listen("tcp", clientListener.String())
	if err != nil {
		log.Fatalf("could not open a TCP connection: %s", err)
	}
	log.Printf("== LISTENER STARTED ON %s ==\n", clientListener.String())

	// Address Updater Listener
	addrLn, err := net.Listen("tcp", addrListener.String())
	if err != nil {
		log.Fatalf("could not setup a TCP listener: %s", err)
        return
	}
	log.Printf("== ADDR LISTENER STARTED ON %s ==\n", addrListener)

	connDone := make(chan net.Addr)

	// Keeps track of number of connected clients, mu is its mutex
	var ongoing int
	var mu sync.Mutex

	// Waits for connection disconnect, then marks the connection free for
	// other connection to connect.
	go func() {
		for addr := range connDone {
			mu.Lock()
			ongoing--
			log.Printf("[%d] CONNECTION ENDED WITH %s\n", ongoing, addr)
			mu.Unlock()
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

			data, err := ioutil.ReadAll(conn)
			if err != nil {
				log.Printf("error reading from %s connected on the address updater endpoint: %s\n", conn.RemoteAddr(), err)
			}

			parsedIP := net.ParseIP(strings.Trim(string(data), "\n"))
			if parsedIP == nil {
				log.Printf("ip received on the updater endpoint is invalid (%s)\n", string(data))
				continue
			}
            
            old := remote.IP
			remote.IP = parsedIP.String()
			log.Printf("[!] REMOTE IP UPDATE %s => %s: ", old, parsedIP)
		}
	}()

	// Listener for incoming clients. It will pass the connection to a
	// separate goroutine if the connection is eligible. If another
	// connection exists, it will end it immediately.
	for {
		conn, err := ln.(*net.TCPListener).Accept()
		if err != nil {
			// log the error and return to listening
			log.Printf("could not accept connection: %s", err)
			continue
		}

		mu.Lock()
		if MAX_CONNS > 0 && ongoing >= MAX_CONNS {
			log.Printf("ended connection with %s since the connection limit has been reached\n", conn.RemoteAddr())
			conn.Close()

			mu.Unlock()
			continue
		}
		ongoing++
		log.Printf("[%d] CONNECTION ESTABLISHED WITH %s\n", ongoing, conn.RemoteAddr())

		mu.Unlock()

		go handleConn(conn, remote, connDone)
	}
}

// Parse flags and return 3 addresses:
// - address which will be used for the listener for the client connection
// - address of the remote server which this app is proxying
// - address which will be used for the listener for the endpoint which will receive remote server address updates
func parseFlags() (*Addr, *Addr, *Addr, int) {
	clientListerSockF := flag.String("c", ":3110", "Socket on which this machine listens for incoming connections, format address:port.")
	remoteSockF := flag.String("r", ":22", "Initial remote address of the remote server.")
	addrUpdateSockF := flag.String("u", ":3111", "Socket which listens for the updates of the IP that this machine is proxying.")
    maxConns := flag.Int("maxConns", 0, "Max number of concurrent client connections, 0 or less means no restriction many.")

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

	return clientListerSock, remoteSock, addrUpdateSock, *maxConns
}

// Handle the connection requested by client.
func handleConn(conn net.Conn, socket *Addr, done chan<- net.Addr) {
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
			log.Printf("outbound connection refused: %s", outbound)
			syncChan <- 1
			return
		}
		defer outbound.Close()

		syncChan <- 1

		reader := bufio.NewReader(outbound)
        for {
            a := make([]byte, 1024)
            r, err := reader.Read(a)
            if err == io.EOF {
                break
            }
            if err != nil {
                log.Printf("could not ready bytes from outbound: %v", err)
                break
            }
            conn.Write(a[:r])
        }
		//reader.WriteTo(conn)

		syncChan <- 1
	}()

	<-syncChan

	// something went wrong, don't make a reader and return
	if outbound == nil {
		return
	}

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
func parseSocket(sock string) (*Addr, error) {
	if sock == "" {
		return nil, errors.New("remote socket not passed")
	}
	parts := strings.Split(sock, ":")

    // if the port hasn't been passed in
    if len(parts) == 1 {
        return &Addr{IP: parts[0], Port: -1}, errors.New("port not passed")
    }

	if len(parts) > 2 {
		return nil, fmt.Errorf("expected 1 or 2 parts (ip and port), got %d", len(parts))
	}

	portInt, err := strconv.Atoi(parts[1])
	if err != nil {
		return nil, fmt.Errorf("port %s isn't a number", parts[1])
	}

	return &Addr{IP: parts[0], Port: portInt}, nil
}
