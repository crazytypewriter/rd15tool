#!/bin/sh

[ -e "/tmp/singbox_patch.log" ] && return 0

echo '#!/bin/sh /etc/rc.common
# Copyright (C) 2006 OpenWrt.org

START=98
USE_PROCD=1
PROG=/data/etc/sing-box/sing-box
CONF=/data/etc/sing-box/config.json
WORKDIR=/data/etc/sing-box

start_service() {
        [ -f "$PROG" ] && {
                procd_open_instance
                procd_set_param command ${PROG} run -c ${CONF} -D ${WORKDIR}
                procd_set_param limits core="unlimited"
                procd_set_param limits nofile="1000000 1000000"
                procd_set_param respawn
                procd_close_instance
        }

}

stop_service() {
        SERVICE_SIG_STOP="KILL"
        service_stop ${PROG}

}' > /etc/init.d/sing-box

chmod +x  /etc/init.d/sing-box

/etc/init.d/sing-box start

echo "singbox enabled" > /tmp/singbox_patch.log