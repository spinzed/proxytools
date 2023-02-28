# proxytools
A small dependency-free toolset for simple HTTP/HTTPS proxying and level 4 TCP reverse proxying.

## Installation
To run this, you'll need the Go tool installed (through [golang.org](https://golang.org)
or your favorite package manager via package `go`). This project utilizes go modules
so Go 1.12+ is required, preferably Go 1.15+.  
To install a tool, run `go get github.com/spinzed/proxytools/cmd/{tool}` where tool is `http-proxy`
or `tcp-reverse-proxy`.  
Alternatively, `git clone` this repo, `cd` into it and run `make` with superuser privileges to
install them all. Or `go build cmd/{tool}` which ones you want.  

## What does this toolset consist of?
### 1) **http-proxy**
A simple HTTP/HTTPS (no SOCKS) forward proxy. In HTTP mode works as a bidirectional proxy
and in HTTPS mode as a layer 4 tunnel proxy. To run in with default settings, just run:

```shell
http-proxy 
```

By default, it listens on "0.0.0.0:3128". You can change this with the -addr flag like this:

```shell
http-proxy -addr="localhost:8080"
```

By default, it listens on for HTTP requests (not HTTPS), to enable HTTPS you need to
pass the -https flag as well as -pem and -key flags which are **absolute** paths to
respective certificate files. An example of such config:

```shell
http-proxy -addr=":8080" -https -pem="/var/cert/proxy.pem" -key="/var/cert/proxy.key"
```

### 2) **tcp-reverse-proxy**
A layer 4 reverse proxy for TCP connections. It listens for incoming TCP connections
by default on port 3110 and forwards them to the server which is by default on
0.0.0.0:22. The specialy of this reverse proxy is that the IP of the server that it's
proxying can be changed. It listens on 0.0.0.0:3110 for IP updates.

#### But why would I want to change the IP of the server?
The idea is that a reverse proxy like this is set up on a machine with a static IP
so that it proxies a machine with dynamic IP. The machine with dynamic IP
would send its IP to the server when it detects it has changed. This makes
possible for the client to connect to the machine with dynamic IP without
needing to know its IP at all. This can replace services like ngrok.  

```
|--------|         |---------------|         |----------------|
| Client | ------> | Reverse Proxy | ------> | Remote Machine |
|        | <------ |  (static IP)  | <------ |  (dynamic IP)  |
|--------|         |---------------|         |----------------|  
```

Command line arguments:
- -c - ip:port of the listener for incoming connections. Default: 0.0.0.0:3110
- -u - ip:port of the listener for IP updates of the remote server. Default: 0.0.0.0:3111
- -r - ip:port of the remote server which this proxy is proxying. Default: 0.0.0.0:22
- -maxConns - max client connections which are permitted at a given moment. Default: unlimited  

Example:  

```shell
tcp-reverse-proxy -c="192.168.1.10:80" -u="0.0.0.0:3000" -r="192.168.1.18:22" -maxConns=10
```

This will create a listener for client connection on 192.168.1.10 on port 80, a
listener for remote machine IP updates on 0.0.0.0 (all intefaces) on port 3000 and
it will set as 192.168.1.18 the initial address of the remote machine and port
22 as the **permanent** port of the remote machine.

## Final Thoughts
Everything is subject to change mostly according to my own needs, that means that
plans of releasing 1.0 are nowhere near. Nevertheless, I'm open to any feedback
to improve this piece of software to fit needs not just myself, but others as well (:

