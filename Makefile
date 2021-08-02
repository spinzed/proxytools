output: $(wildcard **/*.go)
	go build ./cmd/tcp-reverse-proxy
	cp ./tcp-reverse-proxy /usr/bin
	rm tcp-reverse-proxy

	go build ./cmd/http-proxy
	cp ./http-proxy /usr/bin
	rm http-proxy

uninstall:
	rm -f /usr/bin/tcp-reverse-proxy
	rm -f /usr/bin/http-proxy
