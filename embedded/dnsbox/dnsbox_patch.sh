#!/bin/sh

[ -e "/tmp/dnsbox_patch.log" ] && return 0

echo '#!/bin/sh /etc/rc.common
# OpenWrt init script for dns-box (binary in /tmp)

START=99
USE_PROCD=1

TMPDIR=/tmp/dns-box
PROG=${TMPDIR}/dns-box
CONF=/data/dns-box/config.json
BIN_URL="https://github.com/crazytypewriter/rd15tool/releases/latest/download/dns-box"

download_binary() {
    mkdir -p "$TMPDIR"
    if [ ! -x "$PROG" ]; then
        echo "[dns-box] Binary not found, downloading..."

        if command -v curl >/dev/null 2>&1; then
            curl -L -o "$PROG" "$BIN_URL" || return 1
        elif command -v wget >/dev/null 2>&1; then
            wget --no-check-certificate -O "$PROG" "$BIN_URL" || return 1
        else
            echo "[dns-box] Neither curl nor wget found!"
            return 1
        fi

        chmod +x "$PROG"
    else
        echo "[dns-box] Binary already exists in $TMPDIR, skip download."
    fi
}

wait_for_tmp() {
    while [ ! -d /tmp ]; do
        echo "[dns-box] Waiting for /tmp..."
        sleep 1
    done
    mkdir -p "$TMPDIR"
}

wait_for_network() {
    . /lib/functions/network.sh
    network_flush_cache
    network_get_ipaddr ip wan
    while [ -z "$ip" ]; do
        echo "[dns-box] Waiting for network (wan)..."
        sleep 2
        network_flush_cache
        network_get_ipaddr ip wan
    done
    echo "[dns-box] Network is ready: $ip"
}

start_service() {
    wait_for_tmp
    wait_for_network

    download_binary || {
        echo "[dns-box] Failed to download binary!"
        return 1
    }

    procd_open_instance
    procd_set_param command ${PROG} -config ${CONF}
    procd_set_param limits core="unlimited"
    procd_set_param limits nofile="1000000 1000000"
    procd_set_param respawn
    procd_set_param stdout 1
    procd_set_param stderr 1
    procd_close_instance
}

stop_service() {
    SERVICE_SIG_STOP="TERM"
    service_stop ${PROG}
}

boot() {
    start_service
}

start() {
    start_service
}

stop() {
    stop_service
}' > /etc/init.d/dns-box

chmod +x /etc/init.d/dns-box
/etc/init.d/dns-box enable
/etc/init.d/dns-box start

echo "dnsbox enabled" > /tmp/dnsbox_patch.log