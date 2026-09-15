package tunnel

import (
"net"
"testing"
)

func TestYamuxSessionAndStream(t *testing.T) {
serverConn, clientConn := net.Pipe()

errChan := make(chan error, 1)

// Start Server Session
go func() {
serverSess, err := NewServerSession(serverConn)
if err != nil {
errChan <- err
return
}

stream, err := serverSess.Accept()
if err != nil {
errChan <- err
return
}

buf := make([]byte, 5)
n, err := stream.Read(buf)
if err != nil || string(buf[:n]) != "PING" {
t.Errorf("Server received invalid payload: %s", string(buf[:n]))
}

_, _ = stream.Write([]byte("PONG"))
stream.Close()
errChan <- nil
}()

// Start Client Session
clientSess, err := NewClientSession(clientConn)
if err != nil {
t.Fatalf("Failed to create client session: %v", err)
}

stream, err := clientSess.Open()
if err != nil {
t.Fatalf("Failed to open stream: %v", err)
}

if _, err := stream.Write([]byte("PING")); err != nil {
t.Fatalf("Failed to write to stream: %v", err)
}

buf := make([]byte, 5)
n, err := stream.Read(buf)
if err != nil || string(buf[:n]) != "PONG" {
t.Fatalf("Client received invalid response: %s", string(buf[:n]))
}

if err := <-errChan; err != nil {
t.Fatalf("Server side error: %v", err)
}
}
