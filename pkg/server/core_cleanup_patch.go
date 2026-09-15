package server

import (
"log"
)

// Close shuts down all listeners and active client sessions gracefully
func (s *Server) Close() {
s.mu.Lock()
defer s.mu.Unlock()

log.Println("[Server] Shutting down and cleaning up resources...")

// Close all public port listeners
for port, listener := range s.listeners {
if err := listener.Close(); err != nil {
log.Printf("[Server] Error closing listener on port %d: %v", port, err)
}
}

// Close all active client sessions
for clientID, clientSess := range s.clients {
if clientSess.Session != nil {
if err := clientSess.Session.Close(); err != nil {
log.Printf("[Server] Error closing session for client %s: %v", clientID, err)
}
}
}

log.Println("[Server] All resources cleaned up successfully.")
}
