#!/bin/sh

[ -e "/tmp/dnsbox_patch.log" ] && return 0

echo '#!/bin/sh /etc/rc.common
# Copyright (C) 2006 OpenWrt.org

START=98
USE_PROCD=1
PROG=/data/dns-box/dns-box
CONF=/data/dns-box/config.json

start_service() {
        [ -f "$PROG" ] && {
                procd_open_instance
                procd_set_param command ${PROG} -config ${CONF}
                procd_set_param limits core="unlimited"
                procd_set_param limits nofile="1000000 1000000"
                procd_set_param respawn
                procd_set_param stdout 1
                procd_set_param stderr 1
                procd_close_instance

        }

}

stop_service() {
        service_stop ${PROG}

}' > /etc/init.d/dns-box

chmod +x  /etc/init.d/dns-box

/etc/init.d/dns-box start

echo "dnsbox enabled" > /tmp/dnsbox_patch.log