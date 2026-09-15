package tunnel

import (
	"io"
	"net"

	"github.com/golang/snappy"
	"github.com/hashicorp/yamux"
)

type Session struct {
	*yamux.Session
}

func NewServerSession(conn net.Conn) (*Session, error) {
	session, err := yamux.Server(conn, nil)
	if err != nil {
		return nil, err
	}
	return &Session{Session: session}, nil
}

func NewClientSession(conn net.Conn) (*Session, error) {
	session, err := yamux.Client(conn, nil)
	if err != nil {
		return nil, err
	}
	return &Session{Session: session}, nil
}

// Pipe performs bi-directional copy without compression
func Pipe(src net.Conn, dst net.Conn) {
	defer src.Close()
	defer dst.Close()

	done := make(chan struct{}, 2)

	go func() {
		_, _ = io.Copy(src, dst)
		done <- struct{}{}
	}()

	go func() {
		_, _ = io.Copy(dst, src)
		done <- struct{}{}
	}()

	<-done
}

// PipeWithCompress handles bidirectional data stream with optional Snappy compression
func PipeWithCompress(src net.Conn, dst net.Conn, compress bool) {
	if !compress {
		Pipe(src, dst)
		return
	}

	defer src.Close()
	defer dst.Close()

	done := make(chan struct{}, 2)

	go func() {
		writer := snappy.NewBufferedWriter(src)
		reader := snappy.NewReader(dst)
		_, _ = io.Copy(writer, reader)
		writer.Flush()
		done <- struct{}{}
	}()

	go func() {
		writer := snappy.NewBufferedWriter(dst)
		reader := snappy.NewReader(src)
		_, _ = io.Copy(writer, reader)
		writer.Flush()
		done <- struct{}{}
	}()

	<-done
}
