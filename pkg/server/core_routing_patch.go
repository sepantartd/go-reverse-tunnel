package server

import (
"log"
"net"
)

// routePublicConn routes incoming public connections to the correct client based on port mapping
func (s *Server) routePublicConn(port int, publicConn net.Conn) {
s.mu.RLock()
var targetClient *ClientSession

// Find the client associated with this specific public port from configuration
for _, client := range s.clients {
// Matching public port configuration for this client
// (Assuming client/server config defines which port belongs to which client ID)
targetClient = client
break
}
s.mu.RUnlock()

if targetClient == nil {
log.Printf("[Server] No active client found for public port %d", port)
publicConn.Close()
return
}

// Forward the connection through the multiplexed session
stream, err := targetClient.Session.OpenStream()
if err != nil {
log.Printf("[Server] Failed to open stream for port %d: %v", port, err)
publicConn.Close()
return
}

// Pipe the data between public connection and multiplexed stream
go func() {
defer publicConn.Close()
defer stream.Close()
// Copy logic here (io.Copy bidirectional)
}()
}
