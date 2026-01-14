# Audio Integration with Xenith

This document explains how ollama-proxy integrates Xenith's audio processing for real-time speech-to-text (STT) and text-to-speech (TTS) in the Google Meet AI Assistant feature.

## Architecture

```
Chrome (Google Meet)
    ↓
Virtual Audio Devices (PulseAudio)
    ↓
Meeting Audio Bridge (ollama-proxy)
    ↓
┌─────────────────────────────────────────┐
│ Audio Processing Pipeline               │
│                                         │
│  Speaker Audio → parec                  │
│         ↓                               │
│  HTTP POST /v1/audio/transcribe         │
│         ↓                               │
│  Python Audio Service (localhost:5050)  │
│    - Whisper STT (from Xenith)          │
│         ↓                               │
│  Transcribed Text → LLM (Ollama)        │
│    - qwen2.5:0.5b on iGPU               │
│         ↓                               │
│  LLM Response → HTTP POST               │
│         ↓                               │
│  Python Audio Service                   │
│    - Piper TTS (from Xenith)            │
│         ↓                               │
│  Audio PCM → pacat → Virtual Mic        │
│         ↓                               │
│  Chrome Microphone Input                │
└─────────────────────────────────────────┘
```

## Components

### 1. Python Audio Service (`scripts/audio_service.py`)

HTTP service that wraps Xenith's STT and TTS backends:

- **Whisper STT**: OpenAI Whisper via PyTorch (CUDA or CPU)
  - Model: `tiny` (fast, suitable for real-time)
  - Input: Base64-encoded PCM audio (16kHz, mono, s16le)
  - Output: Transcribed text

- **Piper TTS**: Fast neural TTS via ONNX
  - Voice: `en_US-lessac-medium` (fast, good quality)
  - Input: Text string
  - Output: Base64-encoded PCM audio (22kHz, mono, s16le)

**Endpoints:**
- `POST /v1/audio/transcribe` - Speech-to-text
- `POST /v1/audio/synthesize` - Text-to-speech
- `GET /health` - Health check
- `GET /v1/audio/info` - Service information

### 2. Go HTTP Backend (`pkg/backends/audiohttp/audiohttp.go`)

HTTP client that communicates with the Python audio service:

- Implements `backends.Backend` interface
- Converts between Go and Python audio formats
- Base64 encoding/decoding for audio data
- Error handling and logging

### 3. Meeting Audio Bridge (`pkg/device/virtual/meetingbridge.go`)

Orchestrates the full audio processing pipeline:

1. Captures audio from Chrome speaker (via `parec`)
2. Buffers audio chunks (~2 seconds)
3. Sends to STT backend (HTTP audio service)
4. Processes text with LLM backend (Ollama)
5. Sends response to TTS backend (HTTP audio service)
6. Plays audio to virtual mic (via `pacat`)
7. Chrome captures from virtual mic

## Dependencies

### Python Dependencies

Install via `scripts/audio_requirements.txt`:

```bash
flask>=2.3.0              # Web framework
numpy>=1.24.0             # Audio processing
piper-tts>=1.2.0          # Fast TTS
openai-whisper>=20230314  # STT
torch>=2.0.0              # PyTorch for Whisper
torchaudio>=2.0.0         # Audio I/O for PyTorch
```

### External Dependencies

- **Xenith**: Must be cloned to `~/src/Xenith`
  ```bash
  git clone https://github.com/daoneill/Xenith.git ~/src/Xenith
  ```

- **Ollama**: Must have `qwen2.5:0.5b` model installed
  ```bash
  ollama pull qwen2.5:0.5b
  ```

## Setup

### 1. Install Python Dependencies

```bash
cd /home/daoneill/src/ollama-proxy
python3 -m venv venv_audio
source venv_audio/bin/activate
pip install -r scripts/audio_requirements.txt
```

### 2. Clone Xenith

```bash
git clone https://github.com/daoneill/Xenith.git ~/src/Xenith
```

### 3. Install Ollama Model

```bash
ollama pull qwen2.5:0.5b
```

### 4. Start Services

**Option A: Manual (Development)**

Terminal 1 - Audio Service:
```bash
./scripts/start_audio_service.sh
```

Terminal 2 - Ollama-Proxy:
```bash
sudo systemctl restart ollama-proxy
```

**Option B: Systemd (Production)**

```bash
# Install audio service
sudo cp scripts/ollama-proxy-audio.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable ollama-proxy-audio
sudo systemctl start ollama-proxy-audio

# Verify
sudo systemctl status ollama-proxy-audio
sudo systemctl status ollama-proxy
```

## Verification

### 1. Check Audio Service

```bash
# Health check
curl http://localhost:5050/health

# Service info
curl http://localhost:5050/v1/audio/info

# Expected output:
{
  "service": "ollama-proxy Audio Service",
  "backends": {
    "stt": {
      "name": "Whisper",
      "model": "tiny",
      "device": "cuda",  # or "cpu"
      "loaded": true
    },
    "tts": {
      "name": "Piper",
      "voice": "en_US-lessac-medium",
      "loaded": true
    }
  }
}
```

### 2. Check Ollama-Proxy

```bash
# Check logs
sudo journalctl -u ollama-proxy -f

# Look for:
# "Creating HTTP audio backend for STT/TTS"
# "Meeting audio bridge started successfully"
```

### 3. Test STT

```bash
# Record a test audio file (16kHz, mono, s16le)
parec --device=alsa_input.pci-0000_00_1f.3.analog-stereo \
      --format=s16le --rate=16000 --channels=1 \
      --file-format=raw > test_audio.raw
# Stop after a few seconds with Ctrl+C

# Encode to base64
AUDIO_B64=$(base64 -w0 test_audio.raw)

# Test transcription
curl -X POST http://localhost:5050/v1/audio/transcribe \
  -H "Content-Type: application/json" \
  -d "{
    \"audio\": \"$AUDIO_B64\",
    \"format\": \"s16le\",
    \"sample_rate\": 16000,
    \"channels\": 1
  }"
```

### 4. Test TTS

```bash
curl -X POST http://localhost:5050/v1/audio/synthesize \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Hello from Piper text to speech",
    "voice": "en_US-lessac-medium",
    "format": "s16le",
    "sample_rate": 22050
  }' | jq -r '.audio' | base64 -d > test_output.raw

# Play the audio
pacat --format=s16le --rate=22050 --channels=1 --file-format=raw test_output.raw
```

## Performance

### Latency Targets

- **STT (Whisper tiny)**: 200-500ms for 2-second audio clip
  - CUDA: ~200ms
  - CPU (i5-11400): ~500ms

- **LLM (qwen2.5:0.5b)**: 100-300ms for short responses
  - iGPU: ~200ms
  - NPU: ~300ms (larger models not supported)

- **TTS (Piper)**: 50-100ms per sentence
  - Piper is CPU-only but extremely fast

- **Total Pipeline**: 350-900ms end-to-end

### Resource Usage

- **Audio Service**:
  - Memory: ~1-2GB (Whisper models in RAM)
  - CPU: 5-20% during processing, <1% idle
  - GPU (CUDA): 20-40% during STT

- **Ollama-Proxy**:
  - Memory: +10MB for HTTP client
  - CPU: <1% overhead

## Troubleshooting

### Audio Service Won't Start

**Error: "Xenith backends not available"**
```bash
# Check if Xenith is cloned
ls ~/src/Xenith/src/audio/

# If not, clone it
git clone https://github.com/daoneill/Xenith.git ~/src/Xenith
```

**Error: "Failed to load Whisper backend"**
```bash
# Install missing dependencies
pip install openai-whisper torch torchaudio
```

**Error: "Failed to load Piper backend"**
```bash
# Install Piper
pip install piper-tts
```

### Ollama-Proxy Can't Connect to Audio Service

**Error: "audio service not available at http://localhost:5050"**
```bash
# Check if audio service is running
curl http://localhost:5050/health

# If not, start it
./scripts/start_audio_service.sh
```

### Meeting Bridge Errors

**Error: "backend ollama-npu does not support audio-to-text"**

This error means the HTTP audio service is not running. The code has been updated to use the HTTP service instead of Ollama for STT/TTS.

**Fix:**
```bash
# Start audio service
./scripts/start_audio_service.sh

# Restart ollama-proxy
sudo systemctl restart ollama-proxy
```

## Model Management

### Whisper Models

Available models (in order of size/quality):
- `tiny` - 39M params, fastest, real-time suitable ✓
- `base` - 74M params, better quality
- `small` - 244M params, high quality
- `medium` - 769M params, very high quality
- `large` - 1550M params, best quality

To change model, edit `scripts/audio_service.py`:
```python
stt_backend = WhisperSTTBackend(
    model="base",  # Change from "tiny" to "base"
    device="auto",
)
```

### Piper Voices

Available voices:
- `en_US-lessac-medium` - Fast, good quality (default) ✓
- `en_US-ryan-high` - Male voice, higher quality
- `en_GB-alan-medium` - British accent

To change voice, edit `scripts/audio_service.py`:
```python
tts_backend = PiperTTSBackend(
    voice="en_US-ryan-high",  # Change voice
    output_dir="/dev/shm/ollama_tts",
)
```

## Integration Points

### How It Works

1. **Startup** (`cmd/proxy/main.go`):
   - Creates virtual audio devices
   - Registers backends with VirtualDeviceManager
   - Auto-starts meeting bridge for NPU backend

2. **Meeting Bridge Start** (`pkg/device/virtual/manager.go:465`):
   - Creates HTTP audio backend instance
   - Health checks the Python audio service
   - Creates MeetingAudioBridge with STT/LLM/TTS backends
   - Starts audio capture/playback processes

3. **Audio Processing** (`pkg/device/virtual/meetingbridge.go`):
   - Captures from speaker monitor (parec)
   - Buffers 2-second chunks
   - Calls HTTP STT endpoint
   - Processes with LLM
   - Calls HTTP TTS endpoint
   - Plays to virtual mic (pacat)

## Future Improvements

### Performance Optimizations

1. **Streaming STT**: Instead of buffering 2-second chunks, implement streaming transcription
2. **Voice Activity Detection**: Only process audio when speech is detected
3. **TTS Caching**: Cache common responses to reduce latency
4. **Model Quantization**: Use INT8 quantized Whisper models for faster inference

### Feature Additions

1. **Multi-Language Support**: Detect and transcribe non-English languages
2. **Speaker Diarization**: Identify different speakers in the meeting
3. **Background Noise Suppression**: Integrate noise cancellation before STT
4. **Real-time Translation**: Translate between languages in real-time

### Alternative Backends

1. **Faster Whisper**: Use faster-whisper library (4x faster than openai-whisper)
2. **OpenVINO STT**: Use OpenVINO-optimized Whisper on NPU/iGPU
3. **MeloTTS**: Use MeloTTS with NPU for faster TTS
4. **Local LLM**: Use OpenVINO LLM backend instead of Ollama

## References

- [Xenith Project](https://github.com/daoneill/Xenith)
- [OpenAI Whisper](https://github.com/openai/whisper)
- [Piper TTS](https://github.com/rhasspy/piper)
- [Ollama](https://ollama.ai/)
- [PulseAudio Documentation](https://www.freedesktop.org/wiki/Software/PulseAudio/)
