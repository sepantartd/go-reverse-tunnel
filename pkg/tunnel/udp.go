package tunnel

import (
"encoding/binary"
"io"
"net"
)

// ForwardUDPOverTCP converts incoming UDP datagrams into length-prefixed stream over TCP
func ForwardUDPOverTCP(udpConn *net.UDPConn, tcpConn net.Conn) {
defer udpConn.Close()
defer tcpConn.Close()

done := make(chan struct{}, 2)

// UDP -> TCP
go func() {
buf := make([]byte, 65535)
for {
n, _, err := udpConn.ReadFrom(buf)
if err != nil {
break
}

// Write 2-byte length prefix
lengthBuf := make([]byte, 2)
binary.BigEndian.PutUint16(lengthBuf, uint16(n))

if _, err := tcpConn.Write(lengthBuf); err != nil {
break
}
if _, err := tcpConn.Write(buf[:n]); err != nil {
break
}
}
done <- struct{}{}
}()

// TCP -> UDP
go func() {
lengthBuf := make([]byte, 2)
for {
if _, err := io.ReadFull(tcpConn, lengthBuf); err != nil {
break
}
length := binary.BigEndian.Uint16(lengthBuf)

data := make([]byte, length)
if _, err := io.ReadFull(tcpConn, data); err != nil {
break
}

if _, err := udpConn.Write(data); err != nil {
break
}
}
done <- struct{}{}
}()

<-done
}
