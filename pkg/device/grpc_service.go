package device

import (
	"context"
	"fmt"
	"sync"
	"time"

	devicev1 "github.com/daoneill/ollama-proxy/api/proto/device/v1"
	"github.com/godbus/dbus/v5"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCService implements the gRPC DeviceService interface
// It bridges gRPC calls to the D-Bus DeviceManager
type GRPCService struct {
	devicev1.UnimplementedDeviceServiceServer

	deviceManager *DeviceManager
	logger        *zap.Logger

	// Watchers for device events
	watchersMu sync.RWMutex
	watchers   map[string]chan *devicev1.DeviceEvent
}

// NewGRPCService creates a new gRPC device service
func NewGRPCService(dm *DeviceManager) *GRPCService {
	return &GRPCService{
		deviceManager: dm,
		logger:        dm.logger,
		watchers:      make(map[string]chan *devicev1.DeviceEvent),
	}
}

// RegisterDevice registers a new device
func (s *GRPCService) RegisterDevice(ctx context.Context, req *devicev1.RegisterDeviceRequest) (*devicev1.RegisterDeviceResponse, error) {
	s.logger.Info("gRPC RegisterDevice called",
		zap.String("type", req.Type.String()),
		zap.String("path", req.Path),
		zap.String("name", req.Name),
	)

	// Convert gRPC DeviceType to string
	deviceType := deviceTypeToString(req.Type)
	if deviceType == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid device type: %v", req.Type)
	}

	// Convert capabilities to D-Bus format
	caps := make(map[string]dbus.Variant)
	for k, v := range req.Capabilities {
		caps[k] = dbus.MakeVariant(v)
	}

	// Call D-Bus method (use gRPC as sender for external API calls)
	deviceID, dbusErr := s.deviceManager.RegisterDevice(deviceType, req.Path, req.Name, caps, "ie.fio.OllamaProxy.gRPC")
	if dbusErr != nil {
		return nil, status.Errorf(codes.Internal, "failed to register device: %v", dbusErr)
	}

	return &devicev1.RegisterDeviceResponse{
		DeviceId: deviceID,
	}, nil
}

// UnregisterDevice removes a device
func (s *GRPCService) UnregisterDevice(ctx context.Context, req *devicev1.UnregisterDeviceRequest) (*devicev1.UnregisterDeviceResponse, error) {
	s.logger.Info("gRPC UnregisterDevice called",
		zap.String("device_id", req.DeviceId),
	)

	dbusErr := s.deviceManager.UnregisterDevice(req.DeviceId)
	if dbusErr != nil {
		return nil, status.Errorf(codes.Internal, "failed to unregister device: %v", dbusErr)
	}

	return &devicev1.UnregisterDeviceResponse{
		Success: true,
	}, nil
}

// ListDevices lists all devices
func (s *GRPCService) ListDevices(ctx context.Context, req *devicev1.ListDevicesRequest) (*devicev1.ListDevicesResponse, error) {
	s.logger.Debug("gRPC ListDevices called",
		zap.String("filter_type", req.FilterType.String()),
	)

	// Convert filter type
	filterType := ""
	if req.FilterType != devicev1.DeviceType_DEVICE_TYPE_UNSPECIFIED {
		filterType = deviceTypeToString(req.FilterType)
	}

	// Call D-Bus method
	devicesMap, dbusErr := s.deviceManager.ListDevices(filterType)
	if dbusErr != nil {
		return nil, status.Errorf(codes.Internal, "failed to list devices: %v", dbusErr)
	}

	// Convert to protobuf format
	devices := make([]*devicev1.Device, 0, len(devicesMap))
	for _, deviceMap := range devicesMap {
		device := convertDBusDeviceToProto(deviceMap)
		devices = append(devices, device)
	}

	return &devicev1.ListDevicesResponse{
		Devices: devices,
	}, nil
}

// GetDevice gets details about a specific device
func (s *GRPCService) GetDevice(ctx context.Context, req *devicev1.GetDeviceRequest) (*devicev1.GetDeviceResponse, error) {
	s.logger.Debug("gRPC GetDevice called",
		zap.String("device_id", req.DeviceId),
	)

	deviceMap, dbusErr := s.deviceManager.GetDevice(req.DeviceId)
	if dbusErr != nil {
		return nil, status.Errorf(codes.NotFound, "device not found: %v", dbusErr)
	}

	device := convertDBusDeviceToProto(deviceMap)

	return &devicev1.GetDeviceResponse{
		Device: device,
	}, nil
}

// RequestDeviceAccess requests access to a device
func (s *GRPCService) RequestDeviceAccess(ctx context.Context, req *devicev1.RequestDeviceAccessRequest) (*devicev1.RequestDeviceAccessResponse, error) {
	s.logger.Info("gRPC RequestDeviceAccess called",
		zap.String("device_id", req.DeviceId),
		zap.String("client_id", req.ClientId),
	)

	grantID, shmPath, udsPath, dbusErr := s.deviceManager.RequestDeviceAccess(req.DeviceId, req.ClientId, "ie.fio.OllamaProxy.gRPC")
	if dbusErr != nil {
		return nil, status.Errorf(codes.Internal, "failed to request access: %v", dbusErr)
	}

	return &devicev1.RequestDeviceAccessResponse{
		GrantId: grantID,
		ShmPath: shmPath,
		UdsPath: udsPath,
	}, nil
}

// ReleaseDeviceAccess releases access to a device
func (s *GRPCService) ReleaseDeviceAccess(ctx context.Context, req *devicev1.ReleaseDeviceAccessRequest) (*devicev1.ReleaseDeviceAccessResponse, error) {
	s.logger.Info("gRPC ReleaseDeviceAccess called",
		zap.String("device_id", req.DeviceId),
		zap.String("client_id", req.ClientId),
	)

	dbusErr := s.deviceManager.ReleaseDeviceAccess(req.DeviceId, req.ClientId)
	if dbusErr != nil {
		return nil, status.Errorf(codes.Internal, "failed to release access: %v", dbusErr)
	}

	return &devicev1.ReleaseDeviceAccessResponse{
		Success: true,
	}, nil
}

// SubscribeToDevice streams device data (server streaming)
func (s *GRPCService) SubscribeToDevice(req *devicev1.SubscribeToDeviceRequest, stream devicev1.DeviceService_SubscribeToDeviceServer) error {
	s.logger.Info("gRPC SubscribeToDevice called",
		zap.String("device_id", req.DeviceId),
		zap.String("client_id", req.ClientId),
	)

	// TODO: Implement actual streaming when device drivers are ready
	// For now, return unimplemented
	return status.Errorf(codes.Unimplemented, "device streaming not yet implemented (requires Phase 5: Device Drivers)")
}

// DeviceChannel establishes bidirectional streaming
func (s *GRPCService) DeviceChannel(stream devicev1.DeviceService_DeviceChannelServer) error {
	s.logger.Info("gRPC DeviceChannel called")

	// TODO: Implement bidirectional streaming when device drivers are ready
	return status.Errorf(codes.Unimplemented, "device channel not yet implemented (requires Phase 5: Device Drivers)")
}

// WatchDevices streams device events
func (s *GRPCService) WatchDevices(req *devicev1.WatchDevicesRequest, stream devicev1.DeviceService_WatchDevicesServer) error {
	s.logger.Info("gRPC WatchDevices called",
		zap.String("filter_type", req.FilterType.String()),
	)

	// Create watcher channel
	watcherID := fmt.Sprintf("watcher-%d", time.Now().UnixNano())
	eventCh := make(chan *devicev1.DeviceEvent, 100)

	s.watchersMu.Lock()
	s.watchers[watcherID] = eventCh
	s.watchersMu.Unlock()

	defer func() {
		s.watchersMu.Lock()
		delete(s.watchers, watcherID)
		close(eventCh)
		s.watchersMu.Unlock()
	}()

	// Stream events to client
	for {
		select {
		case event, ok := <-eventCh:
			if !ok {
				return nil
			}

			// Apply filter if specified
			if req.FilterType != devicev1.DeviceType_DEVICE_TYPE_UNSPECIFIED {
				if event.Device.Type != req.FilterType {
					continue
				}
			}

			if err := stream.Send(event); err != nil {
				s.logger.Error("Failed to send device event",
					zap.String("watcher_id", watcherID),
					zap.Error(err),
				)
				return err
			}

		case <-stream.Context().Done():
			s.logger.Info("WatchDevices stream cancelled",
				zap.String("watcher_id", watcherID),
			)
			return stream.Context().Err()
		}
	}
}

// Helper: Convert D-Bus device map to protobuf Device
func convertDBusDeviceToProto(deviceMap map[string]dbus.Variant) *devicev1.Device {
	device := &devicev1.Device{}

	if v, ok := deviceMap["ID"]; ok {
		device.Id = v.Value().(string)
	}
	if v, ok := deviceMap["Type"]; ok {
		device.Type = stringToDeviceType(v.Value().(string))
	}
	if v, ok := deviceMap["Name"]; ok {
		device.Name = v.Value().(string)
	}
	if v, ok := deviceMap["Path"]; ok {
		device.Path = v.Value().(string)
	}
	if v, ok := deviceMap["State"]; ok {
		device.State = stringToDeviceState(v.Value().(string))
	}
	if v, ok := deviceMap["RegisteredAt"]; ok {
		if t, ok := v.Value().(time.Time); ok {
			device.RegisteredAt = t.UnixNano()
		}
	}
	if v, ok := deviceMap["LastUsedAt"]; ok {
		if t, ok := v.Value().(time.Time); ok {
			device.LastUsedAt = t.UnixNano()
		}
	}
	if v, ok := deviceMap["Capabilities"]; ok {
		if caps, ok := v.Value().(map[string]dbus.Variant); ok {
			device.Capabilities = make(map[string]string)
			for k, v := range caps {
				device.Capabilities[k] = fmt.Sprintf("%v", v.Value())
			}
		}
	}

	return device
}

// Helper: Convert protobuf DeviceType to string
func deviceTypeToString(dt devicev1.DeviceType) string {
	switch dt {
	case devicev1.DeviceType_DEVICE_TYPE_MICROPHONE:
		return "microphone"
	case devicev1.DeviceType_DEVICE_TYPE_CAMERA:
		return "camera"
	case devicev1.DeviceType_DEVICE_TYPE_SCREEN:
		return "screen"
	case devicev1.DeviceType_DEVICE_TYPE_SPEAKER:
		return "speaker"
	case devicev1.DeviceType_DEVICE_TYPE_KEYBOARD:
		return "keyboard"
	case devicev1.DeviceType_DEVICE_TYPE_MOUSE:
		return "mouse"
	default:
		return ""
	}
}

// Helper: Convert string to protobuf DeviceType
func stringToDeviceType(s string) devicev1.DeviceType {
	switch s {
	case "microphone":
		return devicev1.DeviceType_DEVICE_TYPE_MICROPHONE
	case "camera":
		return devicev1.DeviceType_DEVICE_TYPE_CAMERA
	case "screen":
		return devicev1.DeviceType_DEVICE_TYPE_SCREEN
	case "speaker":
		return devicev1.DeviceType_DEVICE_TYPE_SPEAKER
	case "keyboard":
		return devicev1.DeviceType_DEVICE_TYPE_KEYBOARD
	case "mouse":
		return devicev1.DeviceType_DEVICE_TYPE_MOUSE
	default:
		return devicev1.DeviceType_DEVICE_TYPE_UNSPECIFIED
	}
}

// Helper: Convert string to protobuf DeviceState
func stringToDeviceState(s string) devicev1.DeviceState {
	switch s {
	case "available":
		return devicev1.DeviceState_DEVICE_STATE_AVAILABLE
	case "in-use":
		return devicev1.DeviceState_DEVICE_STATE_IN_USE
	case "error":
		return devicev1.DeviceState_DEVICE_STATE_ERROR
	case "offline":
		return devicev1.DeviceState_DEVICE_STATE_OFFLINE
	default:
		return devicev1.DeviceState_DEVICE_STATE_UNSPECIFIED
	}
}

// NotifyDeviceEvent sends a device event to all watchers
// This should be called by DeviceManager when devices change
func (s *GRPCService) NotifyDeviceEvent(event *devicev1.DeviceEvent) {
	s.watchersMu.RLock()
	defer s.watchersMu.RUnlock()

	for watcherID, ch := range s.watchers {
		select {
		case ch <- event:
			// Event sent successfully
		default:
			s.logger.Warn("Watcher channel full, dropping event",
				zap.String("watcher_id", watcherID),
			)
		}
	}
}
