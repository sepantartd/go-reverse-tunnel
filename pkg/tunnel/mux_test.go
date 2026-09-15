package tunnel

import (
"net"
"testing"
)

func TestMuxStream(t *testing.T) {
serverConn, clientConn := net.Pipe()

go func() {
serverSess, _ := NewServerSession(serverConn)
stream, _ := serverSess.Accept()
buf := make([]byte, 4)
stream.Read(buf)
stream.Write([]byte("PONG"))
stream.Close()
}()

clientSess, _ := NewClientSession(clientConn)
stream, _ := clientSess.Open()
stream.Write([]byte("PING"))

buf := make([]byte, 4)
stream.Read(buf)
if string(buf) != "PONG" {
t.Fatalf("Expected PONG, got %s", string(buf))
}
}
