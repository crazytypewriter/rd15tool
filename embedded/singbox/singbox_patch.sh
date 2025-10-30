#!/bin/sh

[ -e "/tmp/singbox_patch.log" ] && return 0

echo '#!/bin/sh /etc/rc.common
# OpenWrt init script for sing-box (binary in /tmp)

START=99
USE_PROCD=1

TMPDIR=/tmp/sing-box
PROG=${TMPDIR}/sing-box
CONF=/data/sing-box/config.json
BIN_URL="https://github.com/crazytypewriter/rd15tool/releases/latest/download/sing-box"

download_binary() {
    mkdir -p "$TMPDIR"
    if [ ! -x "$PROG" ]; then
        echo "[init] Binary not found, downloading..."

        if command -v curl >/dev/null 2>&1; then
            curl -L -o "$PROG" "$BIN_URL" || return 1
        elif command -v wget >/dev/null 2>&1; then
            wget --no-check-certificate -O "$PROG" "$BIN_URL" || return 1
        else
            echo "[init] Neither curl nor wget found!"
            return 1
        fi

        chmod +x "$PROG"
    else
        echo "[init] Binary already exists in /tmp, skip download."
    fi
}

wait_for_tmp() {
    while [ ! -d /tmp ]; do
        echo "[init] Waiting for /tmp..."
        sleep 1
    done
    mkdir -p "$TMPDIR"
}

wait_for_network() {
    . /lib/functions/network.sh
    network_flush_cache
    network_get_ipaddr ip wan
    while [ -z "$ip" ]; do
        echo "[init] Waiting for network (wan)..."
        sleep 2
        network_flush_cache
        network_get_ipaddr ip wan
    done
    echo "[init] Network is ready: $ip"
}

start_service() {
    wait_for_tmp
    wait_for_network

    download_binary || {
        echo "[init] Failed to download binary!"
        return 1
    }

    procd_open_instance
    procd_set_param command ${PROG} run -c ${CONF} -D ${TMPDIR}
    procd_set_param limits core="unlimited"
    procd_set_param limits nofile="1000000 1000000"
    procd_set_param respawn
    procd_set_param stdout 1
    procd_set_param stderr 1
    procd_close_instance

    (
        for i in $(seq 1 50); do
            if [ -d "/sys/class/net/tun0" ]; then
                sleep 2
                ip route add default dev tun0 table vpn
                ip -6 route add default dev tun0 table vpn
                exit 0
            fi
            sleep 1
        done
    ) &
}

stop_service() {
        SERVICE_SIG_STOP="KILL"
        service_stop ${PROG}

}' > /etc/init.d/sing-box

chmod +x  /etc/init.d/sing-box

/etc/init.d/sing-box enable

/etc/init.d/sing-box start

echo "singbox enabled" > /tmp/singbox_patch.log