# tcp-proxy
A simple, dependency-free proxy for TCP connections.

## What is this exactly and how does it work?
Well, a (reverse) proxy for TCP connections. It listens for incoming TCP connections
(by default on port 3110) and forwards them to the particular server (by
default on 0.0.0.0:22). IP and port of the server can be passed via command
line arguments or by sending the new it dynamically to a listener (which is
by default listening on 0.0.0.0:3111).

## So, what can this be used for?
The idea is that a reverse proxy like this is set up on a machine with a static IP
so that it proxies a machine with dynamic IP. The machine with dynamic IP
would send its IP to the server when it detects it has changed. This makes
possible for the client to connect to the machine with dynamic IP without
needing to know its IP at all.  

```
|--------|         |---------------|         |----------------|
| Client | ------> | Reverse Proxy | ------> | Remote Machine |
|        | <------ |  (static IP)  | <------ |  (dynamic IP)  |
|--------|         |---------------|         |----------------|  
```

How you use it it's completely up to you. I needed a way to ssh into my machine with
a dynamic IP and I didn't like services like ngrok (that's why the default port of
the remote machine is 22). Your use case may be same or entirely different.

## How can I run this?
To run this, you'll need the Go tool installed (through [golang.org](https://golang.org)
or your favorite package manager via package `go`). This project utilizes go modules
so Go 1.12+ is required, preferably Go 1.15+.  
To install this tool you can run `go install`, this way you can use it anywhere
on the system via command `tcp-proxy`. Installing this it is not required, it is
possible to run it using `go run .` in the source code directory.  

Running this tool without passing any arguments will use these default settings:
- the listener for client connections will listen on 0.0.0.0:3110. By default,
there isn't limit to how many connection can be maintained at a given moment, but
that can be changed with the -maxConns flag
- the listener for address updates will listen on 0.0.0.0:3111
- on client connection, the proxy will forward the connection to 0.0.0.0:22. Dynamic
IP updates will change the remote IP if the passed IP is valid. The port cannot be
changed during runtime.  
To change these settings, command line arguments should be passed like this:

```shell
tcp-proxy -c="192.168.1.10:80" -u="0.0.0.0:3000" -r="192.168.1.18:22" -maxConns=10
```

or with `go run`: 

```shell
go run . -c="192.168.1.10:80" -u="0.0.0.0:3000" -r="192.168.1.18:22" -maxConns=10
```

This will create a listener for client connection on 192.168.1.10 on port 80, a
listener for remote machine IP updates on 0.0.0.0 (all intefaces) on port 3000 and
it will set as 192.168.1.18 the initial address of the remote machine and port
22 as the **permanent** port of the remote machine.

## Final Thoughts
Everything is subject to change mostly according to my own needs, that means that
plans of releasing 1.0 are nowhere near. Nevertheless, I'm open to any feedback
to improve this piece of software to fit needs not just myself, but others as well (:

