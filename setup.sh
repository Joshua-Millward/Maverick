#!/bin/bash

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

function error_exit {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

function info_msg {
    echo -e "${GREEN}[+]${NC} $1"
}

function warning_msg {
    echo -e "${YELLOW}[!]${NC} $1"
}

ADAPTIX_DIR=""
AGENT_NAME="maverick"
LISTENER_NAME="maverick_listener"

while [[ $# -gt 0 ]]; do
    case $1 in
        --ax)
            ADAPTIX_DIR="$(realpath "$2" 2>/dev/null || echo "$2")"
            shift 2
            ;;
        *)
            error_exit "Unknown parameter: $1"
            ;;
    esac
done

if [ -z "$ADAPTIX_DIR" ]; then
    echo "Usage: $0 --ax <AdaptixC2_directory>"
    echo ""
    echo "Example:"
    echo "  $0 --ax ../AdaptixC2"
    error_exit "Required parameters missing"
fi

if [ ! -d "$ADAPTIX_DIR" ]; then
    error_exit "Directory does not exist: $ADAPTIX_DIR"
fi

if [ ! -d "$ADAPTIX_DIR/AdaptixServer" ]; then
    error_exit "Directory structure incomplete. AdaptixServer not found in: $ADAPTIX_DIR"
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

mkdir -p "$ADAPTIX_DIR/AdaptixServer/extenders" || error_exit "Failed to create extenders directory"
mkdir -p "$ADAPTIX_DIR/dist/extenders" || error_exit "Failed to create dist/extenders directory"

# ================================================
# Clean previous installation
# ================================================
rm -rf "$ADAPTIX_DIR/AdaptixServer/extenders/$AGENT_NAME"
rm -rf "$ADAPTIX_DIR/AdaptixServer/extenders/$LISTENER_NAME"
rm -rf "$ADAPTIX_DIR/dist/extenders/$AGENT_NAME"
rm -rf "$ADAPTIX_DIR/dist/extenders/$LISTENER_NAME"
info_msg "Cleaned previous installation"

# ================================================
# Prepare agent extender directory
# ================================================
AGENT_EXT="$ADAPTIX_DIR/AdaptixServer/extenders/$AGENT_NAME"
mkdir -p "$AGENT_EXT"

# Copy Go source files
cp "$SCRIPT_DIR/src_server/"*.go "$AGENT_EXT/" || error_exit "Failed to copy agent Go source"
cp "$SCRIPT_DIR/src_server/go.mod" "$AGENT_EXT/" || error_exit "Failed to copy agent go.mod"
cp "$SCRIPT_DIR/src_server/Makefile" "$AGENT_EXT/" || error_exit "Failed to copy agent Makefile"

# Copy dist files (config.yaml + ax_config.axs)
mkdir -p "$AGENT_EXT/dist"
cp "$SCRIPT_DIR/dist_agent/config.yaml" "$AGENT_EXT/dist/" || error_exit "Failed to copy agent config.yaml"
cp "$SCRIPT_DIR/dist_agent/ax_config.axs" "$AGENT_EXT/dist/" || error_exit "Failed to copy agent ax_config.axs"

# Copy beacon source (needed for runtime builds via AgentGenerateBuild)
cp -r "$SCRIPT_DIR/src_beacon" "$AGENT_EXT/" || error_exit "Failed to copy src_beacon"

info_msg "Prepared agent extender directory"

# ================================================
# Prepare listener extender directory
# ================================================
LISTENER_EXT="$ADAPTIX_DIR/AdaptixServer/extenders/$LISTENER_NAME"
mkdir -p "$LISTENER_EXT"

# Copy Go source files
cp "$SCRIPT_DIR/src_listener/"*.go "$LISTENER_EXT/" || error_exit "Failed to copy listener Go source"
cp "$SCRIPT_DIR/src_listener/go.mod" "$LISTENER_EXT/" || error_exit "Failed to copy listener go.mod"
cp "$SCRIPT_DIR/src_listener/Makefile" "$LISTENER_EXT/" || error_exit "Failed to copy listener Makefile"

# Copy dist files (config.yaml + ax_config.axs)
mkdir -p "$LISTENER_EXT/dist"
cp "$SCRIPT_DIR/dist_listener/config.yaml" "$LISTENER_EXT/dist/" || error_exit "Failed to copy listener config.yaml"
cp "$SCRIPT_DIR/dist_listener/ax_config.axs" "$LISTENER_EXT/dist/" || error_exit "Failed to copy listener ax_config.axs"

info_msg "Prepared listener extender directory"

# ================================================
# Sync dependencies
# ================================================
SERVER_GOMOD="$ADAPTIX_DIR/AdaptixServer/go.mod"
if [ -f "$SERVER_GOMOD" ]; then
    AXC2_VER=$(grep 'github.com/Adaptix-Framework/axc2' "$SERVER_GOMOD" | awk '{print $2}')
    if [ -n "$AXC2_VER" ]; then
        info_msg "Detected server axc2 version: $AXC2_VER"

        for EXT_DIR in "$AGENT_EXT" "$LISTENER_EXT"; do
            if [ -f "$EXT_DIR/go.mod" ]; then
                CUR_VER=$(grep 'github.com/Adaptix-Framework/axc2' "$EXT_DIR/go.mod" | awk '{print $2}')
                if [ "$CUR_VER" != "$AXC2_VER" ]; then
                    info_msg "Updating $(basename $EXT_DIR): axc2 $CUR_VER -> $AXC2_VER"
                    cd "$EXT_DIR" || continue
                    go get "github.com/Adaptix-Framework/axc2@$AXC2_VER" || error_exit "Failed to update axc2"
                    go mod tidy || error_exit "Failed to tidy go.mod"
                fi
            fi
        done
    fi
fi

# ================================================
# Setup Go workspace
# ================================================
cd "$ADAPTIX_DIR/AdaptixServer" || error_exit "Could not enter AdaptixServer"

go work use "extenders/$AGENT_NAME" || error_exit "Failed to add agent to Go workspace"
go work use "extenders/$LISTENER_NAME" || error_exit "Failed to add listener to Go workspace"
go work sync || error_exit "Failed to sync Go workspace"
info_msg "Go workspace configured"

# ================================================
# Build plugins
# ================================================
make -C "extenders/$AGENT_NAME" plugin || error_exit "Failed to build agent plugin"
info_msg "Built agent plugin"

make -C "extenders/$LISTENER_NAME" all || error_exit "Failed to build listener plugin"
info_msg "Built listener plugin"

# ================================================
# Copy dist files
# ================================================
cd "$SCRIPT_DIR"

# Agent dist
AGENT_DIST="$ADAPTIX_DIR/dist/extenders/$AGENT_NAME"
mkdir -p "$AGENT_DIST"
cp -r "$AGENT_EXT/dist"/* "$AGENT_DIST/" || error_exit "Failed to copy agent dist"
cp -r "$AGENT_EXT/src_beacon" "$AGENT_DIST/" || warning_msg "Failed to copy src_beacon to dist"
info_msg "Deployed agent to dist"

# Listener dist
LISTENER_DIST="$ADAPTIX_DIR/dist/extenders/$LISTENER_NAME"
mkdir -p "$LISTENER_DIST"
cp -r "$LISTENER_EXT/dist"/* "$LISTENER_DIST/" || error_exit "Failed to copy listener dist"
info_msg "Deployed listener to dist"

# ================================================
# Update profile.yaml
# ================================================
PROFILE="$ADAPTIX_DIR/dist/profile.yaml"
if [ -f "$PROFILE" ]; then
    AGENT_ENTRY="extenders/$AGENT_NAME/config.yaml"
    LISTENER_ENTRY="extenders/$LISTENER_NAME/config.yaml"

    if ! grep -q "$AGENT_ENTRY" "$PROFILE"; then
        sed -i "/extenders:/a\\    - \"$AGENT_ENTRY\"" "$PROFILE"
        info_msg "Added agent to profile.yaml"
    fi

    if ! grep -q "$LISTENER_ENTRY" "$PROFILE"; then
        sed -i "/extenders:/a\\    - \"$LISTENER_ENTRY\"" "$PROFILE"
        info_msg "Added listener to profile.yaml"
    fi
else
    warning_msg "profile.yaml not found at $PROFILE — skipping profile update"
fi

# ================================================
# Done
# ================================================
info_msg "Installation completed successfully"
echo "================================================================"
echo "Agent:    $AGENT_NAME"
echo "Listener: $LISTENER_NAME"
echo "Location: $ADAPTIX_DIR"
echo "================================================================"
