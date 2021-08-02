package internal

import (
	"io"
	"net"
	"time"
)

func MakeTCPConn(address string) (net.Conn, error) {
    conn, err := net.DialTimeout("tcp", address, DIAL_TIMEOUT*time.Second)
    if err != nil {
        return nil, err
    }
    return conn, nil
}

func CopyData(source io.Reader, dest io.Writer) {
	io.Copy(dest, source)
}

func CopyAndClose(source io.ReadCloser, dest io.WriteCloser) {
	defer source.Close()
	defer dest.Close()
    CopyData(source, dest)
}

