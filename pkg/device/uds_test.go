package device

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/daoneill/ollama-proxy/pkg/logging"
)

func TestUDSDeviceServer_CreateAndStop(t *testing.T) {
	server, err := NewUDSDeviceServer("test-device-1", logging.Logger)
	if err != nil {
		t.Fatalf("Failed to create UDS server: %v", err)
	}

	if server.GetClientCount() != 0 {
		t.Error("Expected 0 clients initially")
	}

	if err := server.Stop(); err != nil {
		t.Errorf("Failed to stop server: %v", err)
	}
}

func TestUDSDeviceServer_ClientConnect(t *testing.T) {
	server, err := NewUDSDeviceServer("test-device-2", logging.Logger)
	if err != nil {
		t.Fatalf("Failed to create UDS server: %v", err)
	}
	defer server.Stop()

	server.Start()

	// Give server time to start
	time.Sleep(10 * time.Millisecond)

	// Connect client
	client, err := ConnectUDSClient(server.GetSocketPath(), logging.Logger)
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	// Give time for server to register client
	time.Sleep(10 * time.Millisecond)

	if server.GetClientCount() != 1 {
		t.Errorf("Expected 1 client, got %d", server.GetClientCount())
	}
}

func TestUDSDeviceServer_MessageExchange(t *testing.T) {
	server, err := NewUDSDeviceServer("test-device-3", logging.Logger)
	if err != nil {
		t.Fatalf("Failed to create UDS server: %v", err)
	}
	defer server.Stop()

	server.Start()
	time.Sleep(10 * time.Millisecond)

	// Connect client
	client, err := ConnectUDSClient(server.GetSocketPath(), logging.Logger)
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	time.Sleep(10 * time.Millisecond)

	// Send control message
	controlMsg := &UDSMessage{
		Type:    MessageTypeControl,
		Payload: []byte("START"),
	}

	if err := client.SendMessage(controlMsg); err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Receive ACK
	response, err := client.ReceiveMessage()
	if err != nil {
		t.Fatalf("Failed to receive response: %v", err)
	}

	if response.Type != MessageTypeAck {
		t.Errorf("Expected ACK message, got type %d", response.Type)
	}

	if string(response.Payload) != "OK" {
		t.Errorf("Expected payload 'OK', got '%s'", response.Payload)
	}
}

func TestUDSDeviceServer_BroadcastFrameNotification(t *testing.T) {
	server, err := NewUDSDeviceServer("test-device-4", logging.Logger)
	if err != nil {
		t.Fatalf("Failed to create UDS server: %v", err)
	}
	defer server.Stop()

	server.Start()
	time.Sleep(10 * time.Millisecond)

	// Connect multiple clients
	client1, err := ConnectUDSClient(server.GetSocketPath(), logging.Logger)
	if err != nil {
		t.Fatalf("Failed to connect client 1: %v", err)
	}
	defer client1.Close()

	client2, err := ConnectUDSClient(server.GetSocketPath(), logging.Logger)
	if err != nil {
		t.Fatalf("Failed to connect client 2: %v", err)
	}
	defer client2.Close()

	time.Sleep(10 * time.Millisecond)

	// Broadcast frame notification
	frameIndex := uint64(42)
	frameSize := uint32(1024)
	server.BroadcastFrameNotification(frameIndex, frameSize)

	// Both clients should receive the notification
	for i, client := range []*UDSClient{client1, client2} {
		msg, err := client.ReceiveMessage()
		if err != nil {
			t.Fatalf("Client %d failed to receive notification: %v", i+1, err)
		}

		if msg.Type != MessageTypeFrameNotify {
			t.Errorf("Client %d: expected FrameNotify, got type %d", i+1, msg.Type)
		}

		if len(msg.Payload) != 12 {
			t.Errorf("Client %d: expected payload size 12, got %d", i+1, len(msg.Payload))
		}

		receivedIndex := binary.LittleEndian.Uint64(msg.Payload[0:8])
		receivedSize := binary.LittleEndian.Uint32(msg.Payload[8:12])

		if receivedIndex != frameIndex {
			t.Errorf("Client %d: expected frame index %d, got %d", i+1, frameIndex, receivedIndex)
		}

		if receivedSize != frameSize {
			t.Errorf("Client %d: expected frame size %d, got %d", i+1, frameSize, receivedSize)
		}
	}
}

func TestUDSDeviceServer_BroadcastMetadata(t *testing.T) {
	server, err := NewUDSDeviceServer("test-device-5", logging.Logger)
	if err != nil {
		t.Fatalf("Failed to create UDS server: %v", err)
	}
	defer server.Stop()

	server.Start()
	time.Sleep(10 * time.Millisecond)

	client, err := ConnectUDSClient(server.GetSocketPath(), logging.Logger)
	if err != nil {
		t.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	time.Sleep(10 * time.Millisecond)

	// Broadcast metadata
	metadata := []byte(`{"fps":30,"resolution":"1920x1080"}`)
	server.BroadcastMetadata(metadata)

	// Receive metadata
	msg, err := client.ReceiveMessage()
	if err != nil {
		t.Fatalf("Failed to receive metadata: %v", err)
	}

	if msg.Type != MessageTypeMetadata {
		t.Errorf("Expected Metadata message, got type %d", msg.Type)
	}

	if string(msg.Payload) != string(metadata) {
		t.Errorf("Metadata mismatch: expected %s, got %s", metadata, msg.Payload)
	}
}

func TestUDSDeviceServer_MultipleClients(t *testing.T) {
	server, err := NewUDSDeviceServer("test-device-6", logging.Logger)
	if err != nil {
		t.Fatalf("Failed to create UDS server: %v", err)
	}
	defer server.Stop()

	server.Start()
	time.Sleep(10 * time.Millisecond)

	// Connect 5 clients
	clients := make([]*UDSClient, 5)
	for i := 0; i < 5; i++ {
		client, err := ConnectUDSClient(server.GetSocketPath(), logging.Logger)
		if err != nil {
			t.Fatalf("Failed to connect client %d: %v", i, err)
		}
		defer client.Close()
		clients[i] = client
	}

	time.Sleep(20 * time.Millisecond)

	if server.GetClientCount() != 5 {
		t.Errorf("Expected 5 clients, got %d", server.GetClientCount())
	}

	// Disconnect one client
	clients[2].Close()
	time.Sleep(10 * time.Millisecond)

	if server.GetClientCount() != 4 {
		t.Errorf("Expected 4 clients after disconnect, got %d", server.GetClientCount())
	}
}

func BenchmarkUDSDeviceServer_SendMessage(b *testing.B) {
	server, err := NewUDSDeviceServer("bench-device", logging.Logger)
	if err != nil {
		b.Fatalf("Failed to create UDS server: %v", err)
	}
	defer server.Stop()

	server.Start()
	time.Sleep(10 * time.Millisecond)

	client, err := ConnectUDSClient(server.GetSocketPath(), logging.Logger)
	if err != nil {
		b.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	time.Sleep(10 * time.Millisecond)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		server.BroadcastFrameNotification(uint64(i), 1024)
		client.ReceiveMessage()
	}

	// Target: 10-50μs latency
}

func BenchmarkUDSClient_SendReceive(b *testing.B) {
	server, err := NewUDSDeviceServer("bench-device-2", logging.Logger)
	if err != nil {
		b.Fatalf("Failed to create UDS server: %v", err)
	}
	defer server.Stop()

	server.Start()
	time.Sleep(10 * time.Millisecond)

	client, err := ConnectUDSClient(server.GetSocketPath(), logging.Logger)
	if err != nil {
		b.Fatalf("Failed to connect client: %v", err)
	}
	defer client.Close()

	time.Sleep(10 * time.Millisecond)

	msg := &UDSMessage{
		Type:    MessageTypeControl,
		Payload: []byte("PING"),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if err := client.SendMessage(msg); err != nil {
			b.Fatal(err)
		}
		if _, err := client.ReceiveMessage(); err != nil {
			b.Fatal(err)
		}
	}
}
