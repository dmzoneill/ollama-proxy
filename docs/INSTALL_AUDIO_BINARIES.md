# Installing Audio Processing Binaries

The ollama-proxy uses native binaries for audio processing:
- **whisper.cpp** - Speech-to-text (STT)
- **piper** - Text-to-speech (TTS)

Both run natively in Go via subprocess calls, with OpenVINO acceleration on NPU/iGPU.

## Installation

### 1. Install Whisper.cpp with OpenVINO Support

```bash
# Clone whisper.cpp
cd ~/src
git clone https://github.com/ggerganov/whisper.cpp.git
cd whisper.cpp

# Build with OpenVINO backend for NPU/iGPU acceleration
cmake -B build -DWHISPER_OPENVINO=ON
cmake --build build -j

# Install binary
sudo cp build/bin/main /usr/local/bin/whisper-cpp
sudo chmod +x /usr/local/bin/whisper-cpp

# Download Whisper tiny model (fast, suitable for real-time)
bash ./models/download-ggml-model.sh tiny

# Move model to expected location
mkdir -p ~/.cache/whisper
cp models/ggml-tiny.bin ~/.cache/whisper/
```

### 2. Install Piper TTS

```bash
# Download pre-built Piper binary
cd /tmp
wget https://github.com/rhasspy/piper/releases/download/v1.2.0/piper_amd64.tar.gz
tar -xzf piper_amd64.tar.gz

# Install binary
sudo cp piper/piper /usr/local/bin/
sudo chmod +x /usr/local/bin/piper

# Download voice model
mkdir -p ~/.cache/piper
cd ~/.cache/piper
wget https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/lessac/medium/en_US-lessac-medium.onnx
wget https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/lessac/medium/en_US-lessac-medium.onnx.json
```

### 3. Verify Installation

```bash
# Test whisper.cpp
whisper-cpp --version

# Test piper
echo "Hello from Piper" | piper --model ~/.cache/piper/en_US-lessac-medium.onnx --output_file /tmp/test.wav
paplay /tmp/test.wav
```

## OpenVINO Acceleration

For NPU/iGPU acceleration, ensure OpenVINO is installed:

```bash
# Check OpenVINO devices
python3 -c "import openvino as ov; print(ov.Core().available_devices)"

# Expected output should include: NPU, GPU.0 (iGPU), CPU
```

## Troubleshooting

### Whisper.cpp not found

**Error**: `exec: "whisper-cpp": executable file not found in $PATH`

**Fix**:
```bash
which whisper-cpp  # Should output /usr/local/bin/whisper-cpp
sudo ln -s /usr/local/bin/whisper-cpp /usr/bin/whisper-cpp
```

### Piper not found

**Error**: `exec: "piper": executable file not found in $PATH`

**Fix**:
```bash
which piper  # Should output /usr/local/bin/piper
sudo ln -s /usr/local/bin/piper /usr/bin/piper
```

### Model files missing

**Error**: `failed to load model`

**Fix**:
```bash
# Whisper model
ls ~/.cache/whisper/ggml-tiny.bin

# Piper model
ls ~/.cache/piper/en_US-lessac-medium.onnx
```

## Performance

### Whisper.cpp (STT)
- **CPU**: ~500ms for 2-second audio clip
- **iGPU (OpenVINO)**: ~300ms for 2-second audio clip
- **NPU (OpenVINO)**: ~400ms for 2-second audio clip

### Piper (TTS)
- **CPU-only**: 50-100ms per sentence
- **Very fast**, no GPU acceleration needed

## Alternative Models

### Whisper Models

```bash
# Download different models (in order of size/quality)
cd ~/src/whisper.cpp
bash ./models/download-ggml-model.sh base    # Better quality, slower
bash ./models/download-ggml-model.sh small   # High quality
bash ./models/download-ggml-model.sh medium  # Very high quality
```

Update `/home/daoneill/src/ollama-proxy/pkg/backends/ollama/ollama.go`:
```go
cmd := exec.CommandContext(ctx, "whisper-cpp",
    "-m", "/home/daoneill/.cache/whisper/ggml-base.bin", // Change model
    "-f", tmpFile,
    "-nt",
    "-otxt",
)
```

### Piper Voices

```bash
# Download different voices
cd ~/.cache/piper

# Male voice (Ryan)
wget https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/ryan/high/en_US-ryan-high.onnx
wget https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_US/ryan/high/en_US-ryan-high.onnx.json

# British accent (Alan)
wget https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_GB/alan/medium/en_GB-alan-medium.onnx
wget https://huggingface.co/rhasspy/piper-voices/resolve/main/en/en_GB/alan/medium/en_GB-alan-medium.onnx.json
```

Update `/home/daoneill/src/ollama-proxy/pkg/backends/ollama/ollama.go`:
```go
cmd := exec.CommandContext(ctx, "piper",
    "--model", "/home/daoneill/.cache/piper/en_US-ryan-high.onnx", // Change voice
    "--output_file", tmpFile,
)
```

## Integration with Ollama-Proxy

Once installed, the binaries are automatically used by ollama-proxy:

1. **STT**: Audio captured from Chrome speaker → whisper.cpp → Text
2. **LLM**: Text → Ollama (qwen2.5:0.5b on iGPU) → Response
3. **TTS**: Response → piper → Audio → Chrome microphone

No Python dependencies required - everything runs natively in Go!
