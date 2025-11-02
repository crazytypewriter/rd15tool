#!/bin/sh

[ -e "/tmp/singbox_patch.log" ] && return 0

cat << 'EOF' > /etc/init.d/sing-box
#!/bin/sh /etc/rc.common
# OpenWrt init script for sing-box (binary in /tmp)

START=99
USE_PROCD=1

TMPDIR=/tmp/sing-box
PROG=${TMPDIR}/sing-box
CONF=/data/sing-box/config.json
BIN_URL="https://github.com/crazytypewriter/rd15tool/releases/latest/download/sing-box"
API_URL="https://api.github.com/repos/crazytypewriter/rd15tool/releases/latest"
VER_FILE=${TMPDIR}/version.txt

wait_for_tmp() {
    while [ ! -d /tmp ]; do
        echo "[sing-box] Waiting for /tmp..."
        sleep 1
    done
    mkdir -p "$TMPDIR"
}

wait_for_network() {
    . /lib/functions/network.sh
    network_flush_cache
    network_get_ipaddr ip wan
    while [ -z "$ip" ]; do
        echo "[sing-box] Waiting for network (wan)..."
        sleep 2
        network_flush_cache
        network_get_ipaddr ip wan
    done
    echo "[sing-box] Network is ready: $ip"
}

download_binary() {
    mkdir -p "$TMPDIR"
    latest_version=$(curl -s --max-time 5 "$API_URL" | awk '/"name": "sing-box"/{f=1} f && /"label":/{gsub(/[",]/,""); print $2; exit}')
    local_version=""
    [ -x "$PROG" ] && local_version=$("$PROG" version 2>/dev/null | head -n1 | awk '{print $NF}')
    [ -z "$local_version" ] && [ -f "$VER_FILE" ] && local_version=$(cat "$VER_FILE")

    if [ -z "$latest_version" ]; then
        echo "[sing-box] Offline mode. Using local version: ${local_version:-unknown}"
        [ -x "$PROG" ] || { echo "[sing-box] No binary available"; return 1; }
        echo "[sing-box] sing-box v${local_version:-unknown}"
        return 0
    fi

    if [ ! -x "$PROG" ]; then
        echo "[sing-box] Binary not found, downloading v${latest_version}..."
    elif [ "$local_version" != "$latest_version" ]; then
        echo "[sing-box] Version mismatch (local: ${local_version:-none}, latest: ${latest_version}), updating..."
        rm -f "$PROG"
    else
        echo "[sing-box] sing-box v$local_version (up to date)"
        return 0
    fi

    if command -v curl >/dev/null 2>&1; then
        curl -L -o "$PROG" "$BIN_URL" || return 1
    elif command -v wget >/dev/null 2>&1; then
        wget --no-check-certificate -O "$PROG" "$BIN_URL" || return 1
    else
        echo "[sing-box] Neither curl nor wget found!"
        return 1
    fi

    chmod +x "$PROG"
    echo "$latest_version" > "$VER_FILE"
    echo "[sing-box] Installed sing-box v$latest_version"
}

start_service() {
    wait_for_tmp
    wait_for_network
    download_binary || { echo "[sing-box] Failed to download binary!"; return 1; }

    procd_open_instance
    procd_set_param command ${PROG} run -c ${CONF} -D ${TMPDIR}
    procd_set_param limits core="unlimited"
    procd_set_param limits nofile="1000000 1000000"
    procd_set_param respawn
    procd_set_param stdout 1
    procd_set_param stderr 1
    procd_close_instance

    ver=$("$PROG" version 2>/dev/null | head -n1 | awk '{print $NF}')
    echo "[sing-box] Started (v${ver:-unknown})"
}

stop_service() {
    pid=$(ps w | grep "$PROG run -c" | grep -v grep | awk '{print $1}')
    if [ -n "$pid" ]; then
        kill -TERM "$pid" 2>/dev/null
        echo "[sing-box] Stopped"
    else
        echo "[sing-box] Not running"
    fi
}

service_triggers() {
    procd_add_reload_trigger "sing-box"
}
EOF

chmod +x /etc/init.d/sing-box
/etc/init.d/sing-box enable
/etc/init.d/sing-box start
echo "singbox enabled" > /tmp/singbox_patch.log
