package server

import (
	"bufio"
	"net"
	"sync"
	"time"
)

type Client struct {
	conn net.Conn
	id   int
}

type Server struct {
	tcpListener   net.Listener
	clients       map[int]*Client
	clientsMutex  sync.RWMutex
	clientCounter int
	comPort       *COMPort
	logger        *Logger
	running       bool
	runningMutex  sync.RWMutex
}

func NewServer() *Server {
	return &Server{
		clients: make(map[int]*Client),
		logger:  NewLogger(1000),
		running: true,
	}
}

func (s *Server) StartTCPServer(addr string) error {
	var err error
	s.tcpListener, err = net.Listen("tcp", addr)
	if err != nil {
		s.logger.Error("Failed to start TCP server: %v", err)
		return err
	}

	s.logger.Info("TCP server started on %s", addr)
	defer s.tcpListener.Close()

	for {
		if !s.isRunning() {
			break
		}

		s.tcpListener.(*net.TCPListener).SetDeadline(time.Now().Add(1 * time.Second))
		conn, err := s.tcpListener.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if s.isRunning() {
				s.logger.Error("Failed to accept connection: %v", err)
			}
			continue
		}

		s.clientsMutex.Lock()
		s.clientCounter++
		client := &Client{
			conn: conn,
			id:   s.clientCounter,
		}
		s.clients[client.id] = client
		s.clientsMutex.Unlock()

		s.logger.Info("Client %d connected from %s", client.id, conn.RemoteAddr())

		go s.handleClient(client)
	}

	return nil
}

func (s *Server) handleClient(client *Client) {
	defer func() {
		client.conn.Close()
		s.clientsMutex.Lock()
		delete(s.clients, client.id)
		s.clientsMutex.Unlock()
		s.logger.Info("Client %d disconnected", client.id)
	}()

	reader := bufio.NewReader(client.conn)
	for {
		if !s.isRunning() {
			return
		}

		client.conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		_, err := reader.ReadString('\n')
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			break
		}
	}
}

func (s *Server) Broadcast(data string) {
	s.clientsMutex.RLock()
	defer s.clientsMutex.RUnlock()

	if len(s.clients) == 0 {
		return
	}

	// Ensure data ends with newline for barcode scanners
	if data[len(data)-1] != '\n' {
		data += "\n"
	}

	for id, client := range s.clients {
		_, err := client.conn.Write([]byte(data))
		if err != nil {
			s.logger.Warning("Failed to send data to client %d: %v", id, err)
			go s.removeClient(id)
		}
	}
}

func (s *Server) removeClient(id int) {
	s.clientsMutex.Lock()
	defer s.clientsMutex.Unlock()
	if client, exists := s.clients[id]; exists {
		client.conn.Close()
		delete(s.clients, id)
	}
}

func (s *Server) GetClientCount() int {
	s.clientsMutex.RLock()
	defer s.clientsMutex.RUnlock()
	return len(s.clients)
}

func (s *Server) GetClients() []map[string]interface{} {
	s.clientsMutex.RLock()
	defer s.clientsMutex.RUnlock()

	clients := make([]map[string]interface{}, 0, len(s.clients))
	for _, client := range s.clients {
		clients = append(clients, map[string]interface{}{
			"id":   client.id,
			"addr": client.conn.RemoteAddr().String(),
		})
	}
	return clients
}

func (s *Server) isRunning() bool {
	s.runningMutex.RLock()
	defer s.runningMutex.RUnlock()
	return s.running
}

func (s *Server) Stop() {
	s.runningMutex.Lock()
	s.running = false
	s.runningMutex.Unlock()

	if s.tcpListener != nil {
		s.tcpListener.Close()
	}

	if s.comPort != nil {
		s.comPort.Close()
	}

	s.clientsMutex.Lock()
	for _, client := range s.clients {
		client.conn.Close()
	}
	s.clients = make(map[int]*Client)
	s.clientsMutex.Unlock()

	s.logger.Info("Server stopped")
}

func (s *Server) GetLogger() *Logger {
	return s.logger
}

func (s *Server) SetCOMPort(com *COMPort) {
	s.comPort = com
}

func (s *Server) GetCOMPort() *COMPort {
	return s.comPort
}
