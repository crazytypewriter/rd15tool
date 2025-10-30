#!/bin/sh

[ -e "/tmp/firewall_patch.log" ] && exit 0

cat > /data/userdisk/appdata/firewall.sh <<'EOF'
#!/bin/sh

reload() {
    if ! ipset list vpn_domains > /dev/null 2>&1; then
        echo "Error: ipset vpn_domains does not exist" >&2
        return 1
    fi

    ip rule list | grep -q "fwmark 0x2 lookup vpn" || ip rule add fwmark 0x2 lookup vpn
    iptables -t mangle -C OUTPUT -m set --match-set vpn_domains dst -j MARK --set-mark 0x2 2>/dev/null || iptables -t mangle -A OUTPUT -m set --match-set vpn_domains dst -j MARK --set-mark 0x2
    iptables -t mangle -C PREROUTING -m set --match-set vpn_domains dst -j MARK --set-mark 0x2 2>/dev/null || iptables -t mangle -A PREROUTING -m set --match-set vpn_domains dst -j MARK --set-mark 0x2
    iptables -C FORWARD -m mark --mark 0x2 -j ACCEPT 2>/dev/null || iptables -I FORWARD -m mark --mark 0x2 -j ACCEPT
    iptables -t nat -C POSTROUTING -o tun0 -j SNAT --to-source 172.16.250.1 2>/dev/null || iptables -t nat -A POSTROUTING -o tun0 -j SNAT --to-source 172.16.250.1

    ip -6 rule list | grep -q "fwmark 0x2 lookup vpn" || ip -6 rule add fwmark 0x2 lookup vpn

    ip6tables -t mangle -C OUTPUT -m set --match-set vpn_domains6  dst -j MARK --set-mark 0x2 2>/dev/null || ip6tables -t mangle -A OUTPUT -m set --match-set vpn_domains6 dst -j MARK --set-mark 0x2
    ip6tables -C FORWARD -m mark --mark 0x2 -j ACCEPT 2>/dev/null || ip6tables -I FORWARD -m mark --mark 0x2 -j ACCEPT
    ip6tables -t mangle -C PREROUTING -m set --match-set vpn_domains6 dst -j MARK --set-mark 0x2 2>/dev/null || ip6tables -t mangle -A PREROUTING -m set --match-set vpn_domains6 dst -j MARK --set-mark 0x2

    return 0
}

case "$1" in
    reload)
        reload
        ;;
    *)
        echo "plugin_firewall: not support cmd: $1"
        ;;
esac
EOF

chmod +x /data/userdisk/appdata/firewall.sh

if /data/userdisk/appdata/firewall.sh reload; then
    echo "firewall enabled" > /tmp/firewall_patch.log
else
    echo "firewall reload failed; not creating /tmp/firewall_patch.log" >&2
    exit 1
fi