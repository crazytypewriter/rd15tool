package services

import (
	"encoding/json"
	"github.com/crazytypewriter/sing-lib/ray2sing/api/testlink"
	"github.com/crazytypewriter/sing-lib/ray2sing/common"
	"github.com/crazytypewriter/sing-lib/ray2sing/converter"
	"github.com/crazytypewriter/sing-lib/ray2sing/option"
	"os"
)

type ConfigManager struct {
	T *testlink.Service
	C *converter.Service
}

func NewConfigManager() *ConfigManager {
	return &ConfigManager{
		T: &testlink.Service{},
		C: &converter.Service{},
	}
}

func (cm *ConfigManager) SaveFile(name string, data []byte) error {
	return os.WriteFile(name, data, 0644)
}

func (cm *ConfigManager) OutboundsCheck(text string) bool {
	return cm.T.TestLink(text)
}

func (cm *ConfigManager) ApplyOutbounds(config []byte, outbounds string) (string, error) {
	var opts option.Options
	err := opts.UnmarshalJSON(config)

	decodeOutbounds, _ := common.DecodeBase64IfNeeded(outbounds)

	convertedOutbounds, err := converter.Outbounds(decodeOutbounds)
	if err != nil {
		return "", err
	}

	opts.Outbounds = append(opts.Outbounds, convertedOutbounds...)

	for _, convertedOutbound := range convertedOutbounds {
		for i := range opts.Outbounds {
			if opts.Outbounds[i].Type == "selector" {
				opts.Outbounds[i].SelectorOptions.Outbounds = append(
					opts.Outbounds[i].SelectorOptions.Outbounds, convertedOutbound.Tag)
			} else if opts.Outbounds[i].Type == "urltest" {
				opts.Outbounds[i].URLTestOptions.Outbounds = append(
					opts.Outbounds[i].URLTestOptions.Outbounds, convertedOutbound.Tag)
			}
		}
	}

	data, err := json.MarshalIndent(opts, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
