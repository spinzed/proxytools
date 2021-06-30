output: *.go
	go build
	cp ./tcp-proxy /bin
	cp ./tcp-proxy /usr/bin
	rm tcp-proxy

uninstall:
	rm -f /bin/tcp-proxy
	rm -f /usr/bin/tcp-proxy
