#!/bin/sh

[ -e "/tmp/singbox_patch.log" ] && return 0

echo '#!/bin/sh /etc/rc.common
# Copyright (C) 2006 OpenWrt.org

START=98
USE_PROCD=1
PROG=/data/sing-box/sing-box
CONF=/data/sing-box/config.json
WORKDIR=/tmp/sing-box

start_service() {
        [ -f "$PROG" ] && {
                procd_open_instance
                procd_set_param command ${PROG} run -c ${CONF} -D ${WORKDIR}
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

}

stop_service() {
        SERVICE_SIG_STOP="KILL"
        service_stop ${PROG}

}' > /etc/init.d/sing-box

chmod +x  /etc/init.d/sing-box

/etc/init.d/sing-box start

echo "singbox enabled" > /tmp/singbox_patch.log