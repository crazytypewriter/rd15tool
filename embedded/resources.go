package embedded

import (
	_ "embed"
	"fmt"
)

//go:embed ssh/ssh_patch.sh
var SshPatch []byte

//go:embed singbox/sing-box
var SingBoxBinary []byte

//go:embed singbox/singboxini
var SingBoxIni []byte

//go:embed singbox/config.json
var SingBoxConfig []byte

//go:embed singbox/singbox_patch.sh
var SingBoxPatch []byte

//go:embed dnsbox/dns-box
var DnsBoxBinary []byte

//go:embed dnsbox/dnsboxini
var DnsBoxIni []byte

//go:embed dnsbox/config.json
var DnsBoxConfig []byte

//go:embed dnsbox/dnsbox_patch.sh
var DnsBoxPatch []byte

//go:embed firewall/fire.sh
var Firewall []byte

//go:embed firewall/firewall_patch.sh
var FirewallPatch []byte

//go:embed img/be3600.png
var RD15Image []byte

//go:embed img/be5000.png
var RD16Image []byte

func GetRouterImage(model string) ([]byte, error) {
	switch model {
	case "RD15":
		return RD15Image, nil
	case "RD16":
		return RD16Image, nil
	default:

		return nil, fmt.Errorf("image for model %s does not exist", model)
	}
}
