package tunnel

import (
"io"
"log"
"net"
"sync"
"time"

"github.com/golang/snappy"
)

// compressedConn wraps a net.Conn with Snappy compression/decompression
type compressedConn struct {
net.Conn
reader *snappy.Reader
writer *snappy.Writer
}

func NewCompressedConn(conn net.Conn) *compressedConn {
return &compressedConn{
Conn:   conn,
reader: snappy.NewReader(conn),
writer: snappy.NewWriter(conn),
}
}

func (c *compressedConn) Read(p []byte) (int, error) {
return c.reader.Read(p)
}

func (c *compressedConn) Write(p []byte) (int, error) {
n, err := c.writer.Write(p)
if err != nil {
return n, err
}
err = c.writer.Flush()
return n, err
}

// Pipe copies data bidirectionally between two connections with Snappy compression and logs statistics.
func Pipe(c1, c2 net.Conn) {
// Wrap both connections with Snappy compression
compC1 := NewCompressedConn(c1)
compC2 := NewCompressedConn(c2)

defer c1.Close()
defer c2.Close()

start := time.Now()
var wg sync.WaitGroup
wg.Add(2)

var sentBytes, recvBytes int64

// c1 -> c2 (Upload)
go func() {
defer wg.Done()
var err error
sentBytes, err = io.Copy(compC2, compC1)
if err != nil && err != io.EOF {
// error handling
}
if tcp, ok := c2.(*net.TCPConn); ok {
tcp.CloseWrite()
}
}()

// c2 -> c1 (Download)
go func() {
defer wg.Done()
var err error
recvBytes, err = io.Copy(compC1, compC2)
if err != nil && err != io.EOF {
// error handling
}
if tcp, ok := c1.(*net.TCPConn); ok {
tcp.CloseWrite()
}
}()

wg.Wait()
duration := time.Since(start)
log.Printf("[Compressed Traffic] Connection closed | Duration: %v | Raw Sent: ~%d bytes | Raw Received: ~%d bytes", 
duration.Round(time.Millisecond), sentBytes, recvBytes)
}
