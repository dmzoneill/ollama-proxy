package device

import (
	"fmt"
	"reflect"
	"syscall"
	"unsafe"

	"go.uber.org/zap"

	"github.com/daoneill/ollama-proxy/pkg/logging"
)

// ioctl performs a device ioctl. arg must be a non-nil pointer.
// reflect.Value.Pointer keeps the pointee live for the syscall without unsafe.
func ioctl(fd, req uintptr, arg any) error {
	ptr := reflect.ValueOf(arg)
	if ptr.Kind() != reflect.Pointer || ptr.IsNil() {
		return fmt.Errorf("ioctl argument must be a non-nil pointer")
	}
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, req, ptr.Pointer())
	if errno != 0 {
		return errno
	}
	return nil
}

// overlayStruct views the leading bytes of b as *T.
// b must be non-empty and aligned for T.
func overlayStruct[T any](b []byte) *T {
	return (*T)(unsafe.Pointer(&b[0])) //nolint:gosec // G103: overlay ioctl or shared-memory ABI struct on mapped bytes
}

func warnErr(msg string, err error, fields ...zap.Field) {
	if err == nil {
		return
	}
	logging.Warn(msg, append([]zap.Field{zap.Error(err)}, fields...)...)
}
