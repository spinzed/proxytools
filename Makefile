output: $(wildcard **/*.go)
	go build ./cmd/tcp-reverse-proxy
	cp ./tcp-reverse-proxy /bin
	cp ./tcp-reverse-proxy /usr/bin
	rm tcp-reverse-proxy

	go build ./cmd/http-proxy
	cp ./http-proxy /bin
	cp ./http-proxy /usr/bin
	rm http-proxy

uninstall:
	rm -f /bin/tcp-reverse-proxy /usr/bin/tcp-reverse-proxy
	rm -f /bin/http-proxy /usr/bin/http-proxy
