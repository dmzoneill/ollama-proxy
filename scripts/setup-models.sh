#!/bin/bash
# Setup script to pull models on all 4 Ollama backend instances

set -e

echo "=================================================="
echo "Ollama Multi-Backend Model Setup"
echo "=================================================="
echo ""

# Check if Ollama backends are running
check_backend() {
    local port=$1
    local name=$2

    if curl -s "http://localhost:$port/api/tags" > /dev/null 2>&1; then
        echo "✅ $name (port $port) is running"
        return 0
    else
        echo "❌ $name (port $port) is NOT running"
        return 1
    fi
}

echo "Checking if all Ollama backends are running..."
echo ""

all_running=true

check_backend 11434 "NPU Backend" || all_running=false
check_backend 11435 "Intel GPU Backend" || all_running=false
check_backend 11436 "NVIDIA GPU Backend" || all_running=false
check_backend 11437 "CPU Backend" || all_running=false

echo ""

if [ "$all_running" = false ]; then
    echo "⚠️  Not all backends are running!"
    echo ""
    echo "Please start all Ollama instances first:"
    echo "  Terminal 1: OLLAMA_HOST=0.0.0.0:11434 ollama serve  # NPU"
    echo "  Terminal 2: OLLAMA_HOST=0.0.0.0:11435 OLLAMA_INTEL_GPU=1 ollama serve  # Intel GPU"
    echo "  Terminal 3: OLLAMA_HOST=0.0.0.0:11436 ollama serve  # NVIDIA"
    echo "  Terminal 4: OLLAMA_HOST=0.0.0.0:11437 OLLAMA_NUM_GPU=0 ollama serve  # CPU"
    echo ""
    exit 1
fi

echo "✅ All backends are running!"
echo ""
echo "=================================================="
echo "Pulling models on each backend..."
echo "=================================================="
echo ""

# NPU Backend (11434) - Tiny models for ultra-low power
echo "📦 NPU Backend (Port 11434) - Tiny Models"
echo "   Max size: 2GB | Power: 3W | Use: Realtime audio"
echo "---------------------------------------------------"

OLLAMA_HOST=http://localhost:11434 ollama pull qwen2.5:0.5b
echo "   ✅ qwen2.5:0.5b pulled"

OLLAMA_HOST=http://localhost:11434 ollama pull qwen2.5:1.5b
echo "   ✅ qwen2.5:1.5b pulled"

OLLAMA_HOST=http://localhost:11434 ollama pull tinyllama:1b
echo "   ✅ tinyllama:1b pulled"

echo ""

# Intel GPU Backend (11435) - Small to medium models
echo "📦 Intel GPU Backend (Port 11435) - Medium Models"
echo "   Max size: 8GB | Power: 12W | Use: General text/code"
echo "---------------------------------------------------"

OLLAMA_HOST=http://localhost:11435 ollama pull llama3:7b
echo "   ✅ llama3:7b pulled"

OLLAMA_HOST=http://localhost:11435 ollama pull mistral:7b
echo "   ✅ mistral:7b pulled"

OLLAMA_HOST=http://localhost:11435 ollama pull qwen2.5:1.5b
echo "   ✅ qwen2.5:1.5b pulled (fallback)"

echo ""

# NVIDIA GPU Backend (11436) - All models including large
echo "📦 NVIDIA GPU Backend (Port 11436) - Large Models"
echo "   Max size: 24GB | Power: 55W | Use: Complex code/analysis"
echo "---------------------------------------------------"

OLLAMA_HOST=http://localhost:11436 ollama pull llama3:70b
echo "   ✅ llama3:70b pulled"

OLLAMA_HOST=http://localhost:11436 ollama pull llama3:7b
echo "   ✅ llama3:7b pulled"

echo ""

# CPU Backend (11437) - Small models fallback
echo "📦 CPU Backend (Port 11437) - Fallback Models"
echo "   Max size: 16GB | Power: 28W | Use: Emergency fallback"
echo "---------------------------------------------------"

OLLAMA_HOST=http://localhost:11437 ollama pull qwen2.5:1.5b
echo "   ✅ qwen2.5:1.5b pulled"

OLLAMA_HOST=http://localhost:11437 ollama pull llama3:7b
echo "   ✅ llama3:7b pulled"

echo ""
echo "=================================================="
echo "✅ All models successfully loaded!"
echo "=================================================="
echo ""

# Show summary
echo "Model Distribution Summary:"
echo "---------------------------------------------------"
echo "NPU (11434):       qwen2.5:0.5b, qwen2.5:1.5b, tinyllama:1b"
echo "Intel GPU (11435): llama3:7b, mistral:7b, qwen2.5:1.5b"
echo "NVIDIA (11436):    llama3:70b, llama3:7b"
echo "CPU (11437):       qwen2.5:1.5b, llama3:7b"
echo ""

# Verify
echo "Verifying models are loaded..."
echo ""

echo "NPU models:"
OLLAMA_HOST=http://localhost:11434 ollama list | grep -E "qwen|tinyllama" || echo "   (No models found)"

echo ""
echo "Intel GPU models:"
OLLAMA_HOST=http://localhost:11435 ollama list | grep -E "llama|mistral|qwen" || echo "   (No models found)"

echo ""
echo "NVIDIA models:"
OLLAMA_HOST=http://localhost:11436 ollama list | grep -E "llama" || echo "   (No models found)"

echo ""
echo "CPU models:"
OLLAMA_HOST=http://localhost:11437 ollama list | grep -E "qwen|llama" || echo "   (No models found)"

echo ""
echo "=================================================="
echo "🚀 Setup complete! You can now start the proxy:"
echo "   ./bin/ollama-proxy"
echo "=================================================="
