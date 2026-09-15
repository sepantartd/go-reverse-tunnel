package client

import (
"context"
"log"
"net"

"github.com/armon/go-socks5"
)

func StartSocks5Server(listenAddr string, dialer func(ctx context.Context, network, addr string) (net.Conn, error)) error {
conf := &socks5.Config{}
if dialer != nil {
conf.Dial = dialer
}

server, err := socks5.New(conf)
if err != nil {
return err
}

log.Printf("[SOCKS5] Local proxy listening on %s", listenAddr)
return server.ListenAndServe("tcp", listenAddr)
}
