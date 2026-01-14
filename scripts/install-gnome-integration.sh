#!/bin/bash

# install-gnome-integration.sh
# Installation script for Ollama Proxy GNOME Desktop Integration

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo -e "${BLUE}============================================${NC}"
echo -e "${BLUE}Ollama Proxy - GNOME Integration Installer${NC}"
echo -e "${BLUE}============================================${NC}"
echo ""

# Check if running on GNOME
if [ "$XDG_CURRENT_DESKTOP" != "GNOME" ]; then
    echo -e "${YELLOW}Warning: Not running GNOME desktop environment${NC}"
    echo -e "${YELLOW}Current desktop: ${XDG_CURRENT_DESKTOP}${NC}"
    echo ""
    read -p "Continue anyway? (y/N) " -n 1 -r
    echo ""
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 1
    fi
fi

# Function to install GSettings schema
install_gsettings_schema() {
    echo -e "${BLUE}[1/5] Installing GSettings schema...${NC}"

    SCHEMA_FILE="$PROJECT_DIR/data/ie.fio.ollamaproxy.gschema.xml"

    if [ ! -f "$SCHEMA_FILE" ]; then
        echo -e "${RED}Error: Schema file not found: $SCHEMA_FILE${NC}"
        exit 1
    fi

    # Try user installation first (no sudo needed)
    USER_SCHEMA_DIR="$HOME/.local/share/glib-2.0/schemas"
    mkdir -p "$USER_SCHEMA_DIR"

    echo "  Copying schema to $USER_SCHEMA_DIR"
    cp "$SCHEMA_FILE" "$USER_SCHEMA_DIR/"

    echo "  Compiling schemas..."
    glib-compile-schemas "$USER_SCHEMA_DIR"

    echo -e "${GREEN}  ✓ GSettings schema installed${NC}"
    echo ""
}

# Function to install desktop entry
install_desktop_entry() {
    echo -e "${BLUE}[2/5] Installing desktop entry...${NC}"

    DESKTOP_FILE="$PROJECT_DIR/data/ie.fio.ollamaproxy.desktop"

    if [ ! -f "$DESKTOP_FILE" ]; then
        echo -e "${RED}Error: Desktop file not found: $DESKTOP_FILE${NC}"
        exit 1
    fi

    DESKTOP_DIR="$HOME/.local/share/applications"
    mkdir -p "$DESKTOP_DIR"

    echo "  Copying desktop entry to $DESKTOP_DIR"
    cp "$DESKTOP_FILE" "$DESKTOP_DIR/"

    # Update desktop database
    if command -v update-desktop-database &> /dev/null; then
        update-desktop-database "$DESKTOP_DIR"
    fi

    echo -e "${GREEN}  ✓ Desktop entry installed${NC}"
    echo ""
}

# Function to install systemd service
install_systemd_service() {
    echo -e "${BLUE}[3/5] Installing systemd user service...${NC}"

    SERVICE_FILE="$PROJECT_DIR/data/ie.fio.ollamaproxy.service"

    if [ ! -f "$SERVICE_FILE" ]; then
        echo -e "${RED}Error: Service file not found: $SERVICE_FILE${NC}"
        exit 1
    fi

    SERVICE_DIR="$HOME/.config/systemd/user"
    mkdir -p "$SERVICE_DIR"

    echo "  Copying service file to $SERVICE_DIR"
    cp "$SERVICE_FILE" "$SERVICE_DIR/"

    # Reload systemd daemon
    echo "  Reloading systemd daemon..."
    systemctl --user daemon-reload

    # Ask if user wants to enable auto-start
    read -p "  Enable auto-start on login? (Y/n) " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]] || [[ -z $REPLY ]]; then
        systemctl --user enable ie.fio.ollamaproxy.service
        echo -e "${GREEN}  ✓ Auto-start enabled${NC}"
    fi

    # Ask if user wants to start now
    read -p "  Start service now? (Y/n) " -n 1 -r
    echo ""
    if [[ $REPLY =~ ^[Yy]$ ]] || [[ -z $REPLY ]]; then
        systemctl --user start ie.fio.ollamaproxy.service
        sleep 2
        if systemctl --user is-active --quiet ie.fio.ollamaproxy.service; then
            echo -e "${GREEN}  ✓ Service started successfully${NC}"
        else
            echo -e "${YELLOW}  ⚠ Service failed to start. Check logs with:${NC}"
            echo "    journalctl --user -u ie.fio.ollamaproxy.service"
        fi
    fi

    echo -e "${GREEN}  ✓ Systemd service installed${NC}"
    echo ""
}

# Function to install GNOME Shell extension
install_gnome_extension() {
    echo -e "${BLUE}[4/5] Installing GNOME Shell extension...${NC}"

    EXT_SOURCE="$PROJECT_DIR/gnome-shell-extension"

    if [ ! -d "$EXT_SOURCE" ]; then
        echo -e "${RED}Error: Extension directory not found: $EXT_SOURCE${NC}"
        exit 1
    fi

    # Check required files
    REQUIRED_FILES=("metadata.json" "extension.js" "prefs.js" "stylesheet.css")
    for file in "${REQUIRED_FILES[@]}"; do
        if [ ! -f "$EXT_SOURCE/$file" ]; then
            echo -e "${RED}Error: Missing extension file: $file${NC}"
            exit 1
        fi
    done

    EXT_DIR="$HOME/.local/share/gnome-shell/extensions/ollamaproxy@anthropic.com"

    echo "  Creating extension directory: $EXT_DIR"
    mkdir -p "$EXT_DIR"

    echo "  Copying extension files..."
    cp -r "$EXT_SOURCE"/* "$EXT_DIR/"

    echo -e "${GREEN}  ✓ Extension installed${NC}"
    echo ""
}

# Function to enable and restart GNOME Shell
enable_extension() {
    echo -e "${BLUE}[5/5] Enabling GNOME Shell extension...${NC}"

    # Enable the extension
    if command -v gnome-extensions &> /dev/null; then
        gnome-extensions enable ollamaproxy@anthropic.com 2>/dev/null || true
        echo -e "${GREEN}  ✓ Extension enabled${NC}"
    else
        echo -e "${YELLOW}  ⚠ gnome-extensions command not found${NC}"
        echo "    Enable manually via GNOME Extensions app"
    fi

    echo ""
    echo -e "${YELLOW}⚠ IMPORTANT: Restart GNOME Shell to load the extension${NC}"
    echo ""
    echo "  On Wayland: Log out and log back in"
    echo "  On X11: Press Alt+F2, type 'r', and press Enter"
    echo ""
}

# Function to verify installation
verify_installation() {
    echo -e "${BLUE}Verifying installation...${NC}"
    echo ""

    # Check GSettings schema
    if gsettings list-schemas | grep -q "ie.fio.ollamaproxy"; then
        echo -e "${GREEN}  ✓ GSettings schema: OK${NC}"
    else
        echo -e "${RED}  ✗ GSettings schema: NOT FOUND${NC}"
    fi

    # Check desktop entry
    if [ -f "$HOME/.local/share/applications/ie.fio.ollamaproxy.desktop" ]; then
        echo -e "${GREEN}  ✓ Desktop entry: OK${NC}"
    else
        echo -e "${RED}  ✗ Desktop entry: NOT FOUND${NC}"
    fi

    # Check systemd service
    if [ -f "$HOME/.config/systemd/user/ie.fio.ollamaproxy.service" ]; then
        echo -e "${GREEN}  ✓ Systemd service: OK${NC}"
    else
        echo -e "${RED}  ✗ Systemd service: NOT FOUND${NC}"
    fi

    # Check extension
    if [ -d "$HOME/.local/share/gnome-shell/extensions/ollamaproxy@anthropic.com" ]; then
        echo -e "${GREEN}  ✓ GNOME extension: OK${NC}"
    else
        echo -e "${RED}  ✗ GNOME extension: NOT FOUND${NC}"
    fi

    echo ""
}

# Main installation flow
main() {
    install_gsettings_schema
    install_desktop_entry
    install_systemd_service
    install_gnome_extension
    enable_extension
    verify_installation

    echo -e "${GREEN}============================================${NC}"
    echo -e "${GREEN}Installation Complete!${NC}"
    echo -e "${GREEN}============================================${NC}"
    echo ""
    echo "Next steps:"
    echo ""
    echo "1. Restart GNOME Shell (see instructions above)"
    echo "2. Look for the Ollama Proxy indicator in your top bar"
    echo "3. Click the indicator to change efficiency modes"
    echo "4. Check service status:"
    echo "   systemctl --user status ie.fio.ollamaproxy.service"
    echo ""
    echo "Useful commands:"
    echo "  View logs:    journalctl --user -u ie.fio.ollamaproxy.service -f"
    echo "  Stop service: systemctl --user stop ie.fio.ollamaproxy.service"
    echo "  Restart:      systemctl --user restart ie.fio.ollamaproxy.service"
    echo ""
    echo "Configure extension preferences:"
    echo "  Open GNOME Extensions app → Ollama Proxy Indicator → Settings"
    echo ""
}

# Run main installation
main
