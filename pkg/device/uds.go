package device

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"sync"

	"go.uber.org/zap"
)

// MessageType defines the type of UDS message
type MessageType byte

const (
	MessageTypeMetadata      MessageType = 0x01
	MessageTypeControl       MessageType = 0x02
	MessageTypeFrameNotify   MessageType = 0x03
	MessageTypeError         MessageType = 0x04
	MessageTypeAck           MessageType = 0x05
)

// UDSMessage represents a message sent over Unix Domain Socket
type UDSMessage struct {
	Type    MessageType
	Payload []byte
}

// UDSDeviceServer manages Unix Domain Socket connections for a device
type UDSDeviceServer struct {
	socketPath string
	deviceID   string
	listener   net.Listener
	clients    map[net.Conn]*ClientConn
	mu         sync.RWMutex
	logger     *zap.Logger
	ctx        context.Context
	cancel     context.CancelFunc
}

// ClientConn represents a connected client
type ClientConn struct {
	conn     net.Conn
	clientID string
	writeMu  sync.Mutex
}

// NewUDSDeviceServer creates a new Unix Domain Socket server for a device
func NewUDSDeviceServer(deviceID string, logger *zap.Logger) (*UDSDeviceServer, error) {
	socketPath := fmt.Sprintf("/tmp/ollama-proxy-%s.sock", deviceID)

	// Remove existing socket if present
	os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create UDS listener: %w", err)
	}

	// Set permissions
	if err := os.Chmod(socketPath, 0660); err != nil {
		listener.Close()
		return nil, fmt.Errorf("failed to set socket permissions: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	server := &UDSDeviceServer{
		socketPath: socketPath,
		deviceID:   deviceID,
		listener:   listener,
		clients:    make(map[net.Conn]*ClientConn),
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
	}

	logger.Info("UDS server created",
		zap.String("device_id", deviceID),
		zap.String("socket_path", socketPath),
	)

	return server, nil
}

// Start begins accepting client connections
func (s *UDSDeviceServer) Start() {
	go s.acceptLoop()
}

// acceptLoop accepts incoming client connections
func (s *UDSDeviceServer) acceptLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return
			default:
				s.logger.Error("Failed to accept connection", zap.Error(err))
				continue
			}
		}

		s.logger.Info("Client connected",
			zap.String("device_id", s.deviceID),
			zap.String("remote", conn.RemoteAddr().String()),
		)

		// Add client
		client := &ClientConn{
			conn:     conn,
			clientID: conn.RemoteAddr().String(),
		}

		s.mu.Lock()
		s.clients[conn] = client
		s.mu.Unlock()

		// Handle client in goroutine
		go s.handleClient(client)
	}
}

// handleClient processes messages from a client
func (s *UDSDeviceServer) handleClient(client *ClientConn) {
	defer func() {
		s.mu.Lock()
		delete(s.clients, client.conn)
		s.mu.Unlock()

		client.conn.Close()
		s.logger.Info("Client disconnected",
			zap.String("device_id", s.deviceID),
			zap.String("client_id", client.clientID),
		)
	}()

	for {
		msg, err := s.readMessage(client.conn)
		if err != nil {
			if err != io.EOF {
				s.logger.Error("Failed to read message",
					zap.String("client_id", client.clientID),
					zap.Error(err),
				)
			}
			return
		}

		// Process message
		s.processClientMessage(client, msg)
	}
}

// processClientMessage handles a message from a client
func (s *UDSDeviceServer) processClientMessage(client *ClientConn, msg *UDSMessage) {
	switch msg.Type {
	case MessageTypeControl:
		s.logger.Debug("Received control message",
			zap.String("client_id", client.clientID),
			zap.Int("payload_size", len(msg.Payload)),
		)
		// TODO: Process control commands
		// For now, just acknowledge
		s.sendMessage(client, &UDSMessage{
			Type:    MessageTypeAck,
			Payload: []byte("OK"),
		})

	case MessageTypeMetadata:
		s.logger.Debug("Received metadata request",
			zap.String("client_id", client.clientID),
		)
		// TODO: Send device metadata
		// For now, send placeholder
		s.sendMessage(client, &UDSMessage{
			Type:    MessageTypeMetadata,
			Payload: []byte(`{"status":"active"}`),
		})

	default:
		s.logger.Warn("Unknown message type",
			zap.String("client_id", client.clientID),
			zap.Uint8("type", uint8(msg.Type)),
		)
	}
}

// BroadcastFrameNotification notifies all clients that a new frame is available
func (s *UDSDeviceServer) BroadcastFrameNotification(frameIndex uint64, frameSize uint32) {
	// Encode notification: 8 bytes (frameIndex) + 4 bytes (frameSize)
	payload := make([]byte, 12)
	binary.LittleEndian.PutUint64(payload[0:8], frameIndex)
	binary.LittleEndian.PutUint32(payload[8:12], frameSize)

	msg := &UDSMessage{
		Type:    MessageTypeFrameNotify,
		Payload: payload,
	}

	s.mu.RLock()
	clients := make([]*ClientConn, 0, len(s.clients))
	for _, client := range s.clients {
		clients = append(clients, client)
	}
	s.mu.RUnlock()

	// Send to all clients (without holding lock)
	for _, client := range clients {
		if err := s.sendMessage(client, msg); err != nil {
			s.logger.Error("Failed to send frame notification",
				zap.String("client_id", client.clientID),
				zap.Error(err),
			)
		}
	}
}

// BroadcastMetadata sends metadata to all connected clients
func (s *UDSDeviceServer) BroadcastMetadata(metadata []byte) {
	msg := &UDSMessage{
		Type:    MessageTypeMetadata,
		Payload: metadata,
	}

	s.mu.RLock()
	clients := make([]*ClientConn, 0, len(s.clients))
	for _, client := range s.clients {
		clients = append(clients, client)
	}
	s.mu.RUnlock()

	for _, client := range clients {
		if err := s.sendMessage(client, msg); err != nil {
			s.logger.Error("Failed to send metadata",
				zap.String("client_id", client.clientID),
				zap.Error(err),
			)
		}
	}
}

// sendMessage sends a message to a specific client
func (s *UDSDeviceServer) sendMessage(client *ClientConn, msg *UDSMessage) error {
	// Message format:
	// [4 bytes: length] [1 byte: type] [N bytes: payload]

	client.writeMu.Lock()
	defer client.writeMu.Unlock()

	// Write length (payload size + 1 for type)
	length := uint32(len(msg.Payload) + 1)
	if err := binary.Write(client.conn, binary.LittleEndian, length); err != nil {
		return err
	}

	// Write type
	if err := binary.Write(client.conn, binary.LittleEndian, msg.Type); err != nil {
		return err
	}

	// Write payload
	if len(msg.Payload) > 0 {
		if _, err := client.conn.Write(msg.Payload); err != nil {
			return err
		}
	}

	return nil
}

// readMessage reads a message from the connection
func (s *UDSDeviceServer) readMessage(conn net.Conn) (*UDSMessage, error) {
	// Read length
	var length uint32
	if err := binary.Read(conn, binary.LittleEndian, &length); err != nil {
		return nil, err
	}

	if length == 0 || length > 10*1024*1024 { // Max 10MB
		return nil, fmt.Errorf("invalid message length: %d", length)
	}

	// Read type
	var msgType MessageType
	if err := binary.Read(conn, binary.LittleEndian, &msgType); err != nil {
		return nil, err
	}

	// Read payload
	payloadLen := length - 1 // Subtract type byte
	payload := make([]byte, payloadLen)
	if payloadLen > 0 {
		if _, err := io.ReadFull(conn, payload); err != nil {
			return nil, err
		}
	}

	return &UDSMessage{
		Type:    msgType,
		Payload: payload,
	}, nil
}

// GetClientCount returns the number of connected clients
func (s *UDSDeviceServer) GetClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

// GetSocketPath returns the Unix socket path
func (s *UDSDeviceServer) GetSocketPath() string {
	return s.socketPath
}

// Stop stops the UDS server
func (s *UDSDeviceServer) Stop() error {
	s.cancel()

	// Close all client connections
	s.mu.Lock()
	for conn := range s.clients {
		conn.Close()
	}
	s.clients = make(map[net.Conn]*ClientConn)
	s.mu.Unlock()

	// Close listener
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			s.logger.Error("Failed to close listener", zap.Error(err))
		}
	}

	// Remove socket file
	if err := os.Remove(s.socketPath); err != nil && !os.IsNotExist(err) {
		s.logger.Error("Failed to remove socket file",
			zap.String("path", s.socketPath),
			zap.Error(err),
		)
	}

	s.logger.Info("UDS server stopped",
		zap.String("device_id", s.deviceID),
	)

	return nil
}

// UDSClient represents a client connection to a UDS device server
type UDSClient struct {
	conn       net.Conn
	socketPath string
	logger     *zap.Logger
	readMu     sync.Mutex
	writeMu    sync.Mutex
}

// ConnectUDSClient connects to a UDS device server
func ConnectUDSClient(socketPath string, logger *zap.Logger) (*UDSClient, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to UDS: %w", err)
	}

	client := &UDSClient{
		conn:       conn,
		socketPath: socketPath,
		logger:     logger,
	}

	logger.Info("Connected to UDS server", zap.String("socket_path", socketPath))

	return client, nil
}

// SendMessage sends a message to the server
func (c *UDSClient) SendMessage(msg *UDSMessage) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	// Write length
	length := uint32(len(msg.Payload) + 1)
	if err := binary.Write(c.conn, binary.LittleEndian, length); err != nil {
		return err
	}

	// Write type
	if err := binary.Write(c.conn, binary.LittleEndian, msg.Type); err != nil {
		return err
	}

	// Write payload
	if len(msg.Payload) > 0 {
		if _, err := c.conn.Write(msg.Payload); err != nil {
			return err
		}
	}

	return nil
}

// ReceiveMessage receives a message from the server
func (c *UDSClient) ReceiveMessage() (*UDSMessage, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	// Read length
	var length uint32
	if err := binary.Read(c.conn, binary.LittleEndian, &length); err != nil {
		return nil, err
	}

	if length == 0 || length > 10*1024*1024 {
		return nil, fmt.Errorf("invalid message length: %d", length)
	}

	// Read type
	var msgType MessageType
	if err := binary.Read(c.conn, binary.LittleEndian, &msgType); err != nil {
		return nil, err
	}

	// Read payload
	payloadLen := length - 1
	payload := make([]byte, payloadLen)
	if payloadLen > 0 {
		if _, err := io.ReadFull(c.conn, payload); err != nil {
			return nil, err
		}
	}

	return &UDSMessage{
		Type:    msgType,
		Payload: payload,
	}, nil
}

// Close closes the client connection
func (c *UDSClient) Close() error {
	c.logger.Info("Closing UDS client", zap.String("socket_path", c.socketPath))
	return c.conn.Close()
}
