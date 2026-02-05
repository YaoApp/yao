#!/bin/bash
# XFCE desktop configuration script
# Runs on container startup to set up Yao branding and default applications

set -e

XFCE_CONFIG_DIR="$HOME/.config/xfce4"
XFDESKTOP_DIR="$HOME/.config/xfce4/xfconf/xfce-perchannel-xml"
ICONS_DIR="$HOME/.local/share/icons/hicolor"
APPS_DIR="$HOME/.local/share/applications"
DESKTOP_DIR="$HOME/Desktop"

# Create necessary directories
mkdir -p "$XFCE_CONFIG_DIR/panel"
mkdir -p "$XFDESKTOP_DIR"
mkdir -p "$ICONS_DIR/48x48/apps"
mkdir -p "$ICONS_DIR/128x128/apps"
mkdir -p "$ICONS_DIR/256x256/apps"
mkdir -p "$APPS_DIR"
mkdir -p "$DESKTOP_DIR"

# Copy Yao logo to user icons directory
if [ -f /usr/share/yao/yao-logo-48.png ]; then
    cp /usr/share/yao/yao-logo-48.png "$ICONS_DIR/48x48/apps/yao.png"
    cp /usr/share/yao/yao-logo-128.png "$ICONS_DIR/128x128/apps/yao.png"
    cp /usr/share/yao/yao-logo-256.png "$ICONS_DIR/256x256/apps/yao.png"
    # Also copy to system location for panel icon
    sudo cp /usr/share/yao/yao-logo-48.png /usr/share/pixmaps/yao.png 2>/dev/null || true
    gtk-update-icon-cache "$ICONS_DIR" 2>/dev/null || true
fi

# Copy Chromium launcher to applications
if [ -f /usr/share/yao/panel-launcher-chromium.desktop ]; then
    cp /usr/share/yao/panel-launcher-chromium.desktop "$APPS_DIR/chromium-browser.desktop"
fi

# Configure xfdesktop - hide default icons (File System, Home, Trash), keep only custom shortcuts
cat > "$XFDESKTOP_DIR/xfce4-desktop.xml" << 'XMLEOF'
<?xml version="1.0" encoding="UTF-8"?>
<channel name="xfce4-desktop" version="1.0">
  <property name="desktop-icons" type="empty">
    <property name="style" type="int" value="2"/>
    <property name="file-icons" type="empty">
      <property name="show-home" type="bool" value="false"/>
      <property name="show-filesystem" type="bool" value="false"/>
      <property name="show-trash" type="bool" value="false"/>
      <property name="show-removable" type="bool" value="false"/>
    </property>
  </property>
</channel>
XMLEOF

# Copy workspace shortcut to desktop (named "Workspace" with folder icon)
if [ -f /usr/share/yao/workspace.desktop ]; then
    cp /usr/share/yao/workspace.desktop "$DESKTOP_DIR/workspace.desktop"
    chmod +x "$DESKTOP_DIR/workspace.desktop"
fi

# Set Chromium as default browser
xdg-settings set default-web-browser chromium-browser.desktop 2>/dev/null || true

# Configure XFCE panel - set Applications menu icon to Yao logo
# This will be applied when xfce4-panel starts
cat > "$XFDESKTOP_DIR/xfce4-panel.xml" << 'XMLEOF'
<?xml version="1.0" encoding="UTF-8"?>
<channel name="xfce4-panel" version="1.0">
  <property name="configver" type="int" value="2"/>
  <property name="panels" type="array">
    <value type="int" value="1"/>
    <value type="int" value="2"/>
    <property name="dark-mode" type="bool" value="true"/>
    <property name="panel-1" type="empty">
      <property name="position" type="string" value="p=6;x=0;y=0"/>
      <property name="length" type="uint" value="100"/>
      <property name="position-locked" type="bool" value="true"/>
      <property name="icon-size" type="uint" value="16"/>
      <property name="size" type="uint" value="26"/>
      <property name="plugin-ids" type="array">
        <value type="int" value="1"/>
        <value type="int" value="2"/>
        <value type="int" value="3"/>
        <value type="int" value="4"/>
        <value type="int" value="5"/>
        <value type="int" value="6"/>
        <value type="int" value="7"/>
        <value type="int" value="8"/>
        <value type="int" value="9"/>
        <value type="int" value="10"/>
        <value type="int" value="11"/>
      </property>
    </property>
    <property name="panel-2" type="empty">
      <property name="autohide-behavior" type="uint" value="1"/>
      <property name="position" type="string" value="p=10;x=960;y=1054"/>
      <property name="length" type="uint" value="1"/>
      <property name="position-locked" type="bool" value="true"/>
      <property name="size" type="uint" value="48"/>
      <property name="plugin-ids" type="array">
        <value type="int" value="12"/>
        <value type="int" value="13"/>
        <value type="int" value="14"/>
        <value type="int" value="15"/>
        <value type="int" value="16"/>
        <value type="int" value="17"/>
      </property>
    </property>
  </property>
  <property name="plugins" type="empty">
    <property name="plugin-1" type="string" value="applicationsmenu">
      <property name="button-icon" type="string" value="yao"/>
      <property name="button-title" type="string" value=""/>
      <property name="show-button-title" type="bool" value="false"/>
    </property>
    <property name="plugin-2" type="string" value="tasklist">
      <property name="grouping" type="uint" value="1"/>
    </property>
    <property name="plugin-3" type="string" value="separator">
      <property name="expand" type="bool" value="true"/>
      <property name="style" type="uint" value="0"/>
    </property>
    <property name="plugin-4" type="string" value="pager"/>
    <property name="plugin-5" type="string" value="separator">
      <property name="style" type="uint" value="0"/>
    </property>
    <property name="plugin-6" type="string" value="systray">
      <property name="square-icons" type="bool" value="true"/>
    </property>
    <property name="plugin-7" type="string" value="pulseaudio">
      <property name="enable-keyboard-shortcuts" type="bool" value="true"/>
      <property name="show-notifications" type="bool" value="true"/>
    </property>
    <property name="plugin-8" type="string" value="power-manager-plugin"/>
    <property name="plugin-9" type="string" value="notification-plugin"/>
    <property name="plugin-10" type="string" value="separator">
      <property name="style" type="uint" value="0"/>
    </property>
    <property name="plugin-11" type="string" value="clock"/>
    <property name="plugin-12" type="string" value="showdesktop"/>
    <property name="plugin-13" type="string" value="separator"/>
    <property name="plugin-14" type="string" value="launcher">
      <property name="items" type="array">
        <value type="string" value="xfce4-terminal.desktop"/>
      </property>
    </property>
    <property name="plugin-15" type="string" value="launcher">
      <property name="items" type="array">
        <value type="string" value="thunar.desktop"/>
      </property>
    </property>
    <property name="plugin-16" type="string" value="launcher">
      <property name="items" type="array">
        <value type="string" value="chromium-browser.desktop"/>
      </property>
    </property>
    <property name="plugin-17" type="string" value="separator"/>
  </property>
</channel>
XMLEOF

echo "[XFCE Setup] Configuration complete"
