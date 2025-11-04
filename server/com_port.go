package server

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/jacobsa/go-serial/serial"
)

type COMPort struct {
	portName string
	port     io.ReadWriteCloser
	baudRate uint
	ctx      context.Context
	cancel   context.CancelFunc
	running  bool
}

func NewCOMPort(portName string, baudRate uint) *COMPort {
	ctx, cancel := context.WithCancel(context.Background())
	return &COMPort{
		portName: portName,
		baudRate: baudRate,
		ctx:      ctx,
		cancel:   cancel,
		running:  false,
	}
}

func (c *COMPort) Connect() error {
	options := serial.OpenOptions{
		PortName:        c.portName,
		BaudRate:        c.baudRate,
		DataBits:        8,
		StopBits:        1,
		ParityMode:      serial.PARITY_NONE,
		MinimumReadSize: 1,
	}

	port, err := serial.Open(options)
	if err != nil {
		return fmt.Errorf("failed to open COM port %s: %v", c.portName, err)
	}

	c.port = port
	c.running = true
	return nil
}

func (c *COMPort) ReadData(server *Server) {
	if c.port == nil {
		return
	}

	buffer := make([]byte, 128)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			if !c.running {
				return
			}

			n, err := c.port.Read(buffer)
			if err != nil {
				server.logger.Error("Error reading from COM port: %v", err)
				time.Sleep(1 * time.Second)
				continue
			}
			if n > 0 {
				data := make([]byte, n)
				copy(data, buffer[:n])

				// Убираем символы новой строки и возврата каретки
				if len(data) > 0 && (data[len(data)-1] == '\n' || data[len(data)-1] == '\r') {
					data = data[:len(data)-1]
				}
				if len(data) > 0 && (data[len(data)-1] == '\n' || data[len(data)-1] == '\r') {
					data = data[:len(data)-1]
				}

				if len(data) > 0 {
					barcode := string(data)
					server.logger.Info("Barcode scanned: %s", barcode)
					server.Broadcast(barcode)
				}
			}
		}
	}
}

func (c *COMPort) Close() {
	c.running = false
	if c.cancel != nil {
		c.cancel()
	}
	if c.port != nil {
		c.port.Close()
	}
}

func (c *COMPort) IsConnected() bool {
	return c.port != nil && c.running
}

func (c *COMPort) GetPortName() string {
	return c.portName
}

func (c *COMPort) GetBaudRate() uint {
	return c.baudRate
}

// MonitorCOMPort will automatically find and connect to active COM ports
func (s *Server) MonitorCOMPort() {
	for {
		if !s.isRunning() {
			return
		}

		if s.comPort == nil || !s.comPort.IsConnected() {
			ports := s.GetAvailablePorts()
			for _, port := range ports {
				s.logger.Info("Attempting to connect to COM port: %s", port)
				comPort := NewCOMPort(port, 9600) // Standard baud rate for scanners
				if err := comPort.Connect(); err != nil {
					s.logger.Warning("Failed to connect to %s: %v", port, err)
					continue
				}

				s.SetCOMPort(comPort)
				s.logger.Info("Successfully connected to COM port: %s", port)

				// Start reading from the COM port
				go comPort.ReadData(s)
				break
			}

			if s.comPort == nil {
				s.logger.Warning("No active COM ports found, retrying in 5 seconds...")
				time.Sleep(5 * time.Second)
			}
		} else {
			time.Sleep(1 * time.Second)
		}
	}
}

// getAvailablePorts returns a list of available COM ports
// This is a simplified implementation - you might want to enhance it
func (s *Server) GetAvailablePorts() []string {
	availablePorts := []string{}

	// Common COM port names to check
	portsToCheck := []string{}

	// Windows COM ports
	for i := 1; i <= 20; i++ {
		portsToCheck = append(portsToCheck, fmt.Sprintf("COM%d", i))
	}

	// Linux serial ports
	for i := 0; i <= 10; i++ {
		portsToCheck = append(portsToCheck, fmt.Sprintf("/dev/ttyS%d", i))
		portsToCheck = append(portsToCheck, fmt.Sprintf("/dev/ttyUSB%d", i))
		portsToCheck = append(portsToCheck, fmt.Sprintf("/dev/ttyACM%d", i))
	}

	// Test each port to see if it's available
	for _, port := range portsToCheck {
		if s.isPortAvailable(port) {
			availablePorts = append(availablePorts, port)
		}
	}

	return availablePorts
}

// isPortAvailable checks if a specific COM port is available by attempting to open it
func (s *Server) isPortAvailable(portName string) bool {
	options := serial.OpenOptions{
		PortName:        portName,
		BaudRate:        9600, // Use a standard baud rate for testing
		DataBits:        8,
		StopBits:        1,
		ParityMode:      serial.PARITY_NONE,
		MinimumReadSize: 1,
	}

	port, err := serial.Open(options)
	if err != nil {
		return false
	}

	// If we successfully opened it, close it immediately since we were just testing
	port.Close()
	return true
}

// ReconnectCOMPort allows reconnecting to a specific COM port
func (s *Server) ReconnectCOMPort(portName string, baudRate uint) error {
	if s.comPort != nil {
		s.comPort.Close()
		s.comPort = nil
	}

	comPort := NewCOMPort(portName, baudRate)
	if err := comPort.Connect(); err != nil {
		return err
	}

	s.SetCOMPort(comPort)
	s.logger.Info("Reconnected to COM port: %s", portName)
	go comPort.ReadData(s)
	return nil
}
