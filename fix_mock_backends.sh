#!/bin/bash

# This script adds missing multimedia interface methods to all mock backends in test files

cat > /tmp/multimedia_methods.txt << 'EOF'

// Multimedia capability methods (required for backends.Backend interface)
func (m *MockBackend) SupportsAudioToText() bool { return false }
func (m *MockBackend) SupportsTextToAudio() bool { return false }
func (m *MockBackend) SupportsImageToText() bool { return false }
func (m *MockBackend) SupportsTextToImage() bool { return false }
func (m *MockBackend) SupportsVideoToText() bool { return false }
func (m *MockBackend) SupportsTextToVideo() bool { return false }

func (m *MockBackend) TranscribeAudio(ctx context.Context, req *backends.TranscribeRequest) (*backends.TranscribeResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockBackend) TranscribeAudioStream(ctx context.Context, req *backends.TranscribeRequest) (backends.AudioStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockBackend) SynthesizeSpeech(ctx context.Context, req *backends.SynthesizeRequest) (*backends.SynthesizeResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockBackend) SynthesizeSpeechStream(ctx context.Context, req *backends.SynthesizeRequest) (backends.AudioStreamWriter, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockBackend) AnalyzeImage(ctx context.Context, req *backends.ImageAnalysisRequest) (*backends.ImageAnalysisResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockBackend) GenerateImage(ctx context.Context, req *backends.ImageGenRequest) (*backends.ImageGenResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockBackend) GenerateImageStream(ctx context.Context, req *backends.ImageGenRequest) (backends.ImageStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockBackend) AnalyzeVideo(ctx context.Context, req *backends.VideoAnalysisRequest) (*backends.VideoAnalysisResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockBackend) AnalyzeVideoStream(ctx context.Context, req *backends.VideoAnalysisRequest) (backends.VideoStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockBackend) GenerateVideo(ctx context.Context, req *backends.VideoGenRequest) (*backends.VideoGenResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *MockBackend) GenerateVideoStream(ctx context.Context, req *backends.VideoGenRequest) (backends.VideoStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}
EOF

cat > /tmp/multimedia_methods_router.txt << 'EOF'

// Multimedia capability methods (required for backends.Backend interface)
func (m *mockBackendForRouter) SupportsAudioToText() bool { return false }
func (m *mockBackendForRouter) SupportsTextToAudio() bool { return false }
func (m *mockBackendForRouter) SupportsImageToText() bool { return false }
func (m *mockBackendForRouter) SupportsTextToImage() bool { return false }
func (m *mockBackendForRouter) SupportsVideoToText() bool { return false }
func (m *mockBackendForRouter) SupportsTextToVideo() bool { return false }

func (m *mockBackendForRouter) TranscribeAudio(ctx context.Context, req *backends.TranscribeRequest) (*backends.TranscribeResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockBackendForRouter) TranscribeAudioStream(ctx context.Context, req *backends.TranscribeRequest) (backends.AudioStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockBackendForRouter) SynthesizeSpeech(ctx context.Context, req *backends.SynthesizeRequest) (*backends.SynthesizeResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockBackendForRouter) SynthesizeSpeechStream(ctx context.Context, req *backends.SynthesizeRequest) (backends.AudioStreamWriter, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockBackendForRouter) AnalyzeImage(ctx context.Context, req *backends.ImageAnalysisRequest) (*backends.ImageAnalysisResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockBackendForRouter) GenerateImage(ctx context.Context, req *backends.ImageGenRequest) (*backends.ImageGenResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockBackendForRouter) GenerateImageStream(ctx context.Context, req *backends.ImageGenRequest) (backends.ImageStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockBackendForRouter) AnalyzeVideo(ctx context.Context, req *backends.VideoAnalysisRequest) (*backends.VideoAnalysisResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockBackendForRouter) AnalyzeVideoStream(ctx context.Context, req *backends.VideoAnalysisRequest) (backends.VideoStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockBackendForRouter) GenerateVideo(ctx context.Context, req *backends.VideoGenRequest) (*backends.VideoGenResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockBackendForRouter) GenerateVideoStream(ctx context.Context, req *backends.VideoGenRequest) (backends.VideoStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}
EOF

cat > /tmp/multimedia_methods_mockbackend.txt << 'EOF'

// Multimedia capability methods (required for backends.Backend interface)
func (m *mockBackend) SupportsAudioToText() bool { return false }
func (m *mockBackend) SupportsTextToAudio() bool { return false }
func (m *mockBackend) SupportsImageToText() bool { return false }
func (m *mockBackend) SupportsTextToImage() bool { return false }
func (m *mockBackend) SupportsVideoToText() bool { return false }
func (m *mockBackend) SupportsTextToVideo() bool { return false }

func (m *mockBackend) TranscribeAudio(ctx context.Context, req *backends.TranscribeRequest) (*backends.TranscribeResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockBackend) TranscribeAudioStream(ctx context.Context, req *backends.TranscribeRequest) (backends.AudioStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockBackend) SynthesizeSpeech(ctx context.Context, req *backends.SynthesizeRequest) (*backends.SynthesizeResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockBackend) SynthesizeSpeechStream(ctx context.Context, req *backends.SynthesizeRequest) (backends.AudioStreamWriter, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockBackend) AnalyzeImage(ctx context.Context, req *backends.ImageAnalysisRequest) (*backends.ImageAnalysisResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockBackend) GenerateImage(ctx context.Context, req *backends.ImageGenRequest) (*backends.ImageGenResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockBackend) GenerateImageStream(ctx context.Context, req *backends.ImageGenRequest) (backends.ImageStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockBackend) AnalyzeVideo(ctx context.Context, req *backends.VideoAnalysisRequest) (*backends.VideoAnalysisResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockBackend) AnalyzeVideoStream(ctx context.Context, req *backends.VideoAnalysisRequest) (backends.VideoStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockBackend) GenerateVideo(ctx context.Context, req *backends.VideoGenRequest) (*backends.VideoGenResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

func (m *mockBackend) GenerateVideoStream(ctx context.Context, req *backends.VideoGenRequest) (backends.VideoStreamReader, error) {
	return nil, fmt.Errorf("not implemented")
}
EOF

echo "Adding multimedia methods to test files..."

# Fix router tests (mockBackendForRouter)
if grep -q "mockBackendForRouter" pkg/router/forwarding_router_test.go; then
    if ! grep -q "SupportsAudioToText" pkg/router/forwarding_router_test.go; then
        # Find the last method before tests and append
        sed -i '/^\/\/ Test/i\
'"$(cat /tmp/multimedia_methods_router.txt)"'
' pkg/router/forwarding_router_test.go
        echo "✓ Updated pkg/router/forwarding_router_test.go"
    fi
fi

# List of files with mockBackend
files_with_mockbackend=(
    "pkg/classifier/classifier_test.go"
    "pkg/confidence/estimator_test.go"
    "pkg/health/checker_test.go"
    "pkg/dbus/backends_service_test.go"
    "pkg/http/openai/handlers_test.go"
    "pkg/server/server_test.go"
)

for file in "${files_with_mockbackend[@]}"; do
    if [ -f "$file" ]; then
        if ! grep -q "SupportsAudioToText" "$file"; then
            # Find line with "// Tests" or similar and insert before it
            if grep -q "^\/\/ Test" "$file"; then
                sed -i '/^\/\/ Test/i\
'"$(cat /tmp/multimedia_methods_mockbackend.txt)"'
' "$file"
            elif grep -q "^func Test" "$file"; then
                sed -i '/^func Test/i\
'"$(cat /tmp/multimedia_methods_mockbackend.txt)"'
' "$file"
            fi
            echo "✓ Updated $file"
        fi
    fi
done

# Fix websocket tests (also uses MockBackend)
if [ -f "pkg/http/websocket/handler_test.go" ]; then
    if ! grep -q "SupportsAudioToText" "pkg/http/websocket/handler_test.go"; then
        sed -i '/^func Test/i\
'"$(cat /tmp/multimedia_methods.txt)"'
' pkg/http/websocket/handler_test.go
        echo "✓ Updated pkg/http/websocket/handler_test.go"
    fi
fi

rm /tmp/multimedia_methods.txt /tmp/multimedia_methods_router.txt /tmp/multimedia_methods_mockbackend.txt

echo "Done! All mock backends updated."
