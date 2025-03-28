#!/bin/sh

reload() {
  ipset list vpn_domains > /dev/null 2>&1
    if [ $? -ne 0 ]; then
        echo "Warning: ipset 'vpn_domains' does not exist."
        exit 1
    fi

    ip rule list | grep -q "fwmark 0x2 lookup vpn" || ip rule add fwmark 0x2 lookup vpn
    iptables -t mangle -C OUTPUT -m set --match-set vpn_domains dst -j MARK --set-mark 0x2 2>/dev/null || \
        iptables -t mangle -A OUTPUT -m set --match-set vpn_domains dst -j MARK --set-mark 0x2
    iptables -t mangle -C PREROUTING -m set --match-set vpn_domains dst -j MARK --set-mark 0x2 2>/dev/null || \
        iptables -t mangle -A PREROUTING -m set --match-set vpn_domains dst -j MARK --set-mark 0x2
    iptables -C FORWARD -m mark --mark 0x2 -j ACCEPT 2>/dev/null || \
        iptables -I FORWARD -m mark --mark 0x2 -j ACCEPT
    iptables -t nat -C POSTROUTING -o tun0 -j SNAT --to-source 172.16.250.1 2>/dev/null || \
        iptables -t nat -A POSTROUTING -o tun0 -j SNAT --to-source 172.16.250.1
    return
}

case $1 in
    "reload")
        reload
    ;;

    *)
        echo "plugin_firewall: not support cmd: $1"
    ;;
esac