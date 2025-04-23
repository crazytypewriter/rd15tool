#!/bin/sh

[ -e "/tmp/firewall_patch.log" ] && return 0

echo "#!/bin/sh

      reload() {
          if ! ipset list vpn_domains > /dev/null 2>&1; then
              echo \"Error: ipset vpn_domains does not exist\" >&2
              exit 1
          fi

          ip rule list | grep -q \"fwmark 0x2 lookup vpn\" || ip rule add fwmark 0x2 lookup vpn
          iptables -t mangle -C OUTPUT -m set --match-set vpn_domains dst -j MARK --set-mark 0x2 2>/dev/null || iptables -t mangle -A OUTPUT -m set --match-set vpn_domains dst -j MARK --set-mark 0x2
          iptables -t mangle -C PREROUTING -m set --match-set vpn_domains dst -j MARK --set-mark 0x2 2>/dev/null || iptables -t mangle -A PREROUTING -m set --match-set vpn_domains dst -j MARK --set-mark 0x2
          iptables -C FORWARD -m mark --mark 0x2 -j ACCEPT 2>/dev/null || \iptables -I FORWARD -m mark --mark 0x2 -j ACCEPT
          iptables -t nat -C POSTROUTING -o tun0 -j SNAT --to-source 172.16.250.1 2>/dev/null || iptables -t nat -A POSTROUTING -o tun0 -j SNAT --to-source 172.16.250.1

          ip -6 rule list | grep -q \"fwmark 0x2 lookup vpn\" || ip -6 rule add fwmark 0x2 lookup vpn

          ip6tables -t mangle -C OUTPUT -m set --match-set vpn_domains6  dst -j MARK --set-mark 0x2 2>/dev/null || ip6tables -t mangle -A OUTPUT -m set --match-set vpn_domains6 dst -j MARK --set-mark 0x2
          ip6tables -C FORWARD -m mark --mark 0x2 -j ACCEPT 2>/dev/null || ip6tables -I FORWARD -m mark --mark 0x2 -j ACCEPT
          ip6tables -t mangle -C PREROUTING -m set --match-set vpn_domains6 dst -j MARK --set-mark 0x2 2>/dev/null || ip6tables -t mangle -A PREROUTING -m set --match-set vpn_domains6 dst -j MARK --set-mark 0x2
          return
      }

      case \$1 in
          \"reload\")
              reload
          ;;

          *)
              echo \"plugin_firewall: not support cmd: $1\"
          ;;
      esac" > /data/userdisk/appdata/firewall.sh

chmod +x /data/userdisk/appdata/firewall.sh

if /data/userdisk/appdata/firewall.sh reload > /dev/null 2>&1; then
    echo "firewall enabled" > /tmp/firewall_patch.log
fi