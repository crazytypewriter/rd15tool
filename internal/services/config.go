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
	if err != nil {
		return "", err
	}

	decodeOutbounds, _ := common.DecodeBase64IfNeeded(outbounds)

	convertedOutbounds, err := converter.Outbounds(decodeOutbounds)
	if err != nil {
		return "", err
	}

	newOutbounds := make([]option.Outbound, 0, len(convertedOutbounds)+len(opts.Outbounds))
	newOutbounds = append(newOutbounds, convertedOutbounds...)
	newOutbounds = append(newOutbounds, opts.Outbounds...)
	opts.Outbounds = newOutbounds

	for _, convertedOutbound := range convertedOutbounds {
		for i := range opts.Outbounds {
			switch opts.Outbounds[i].Type {
			case "selector":
				newSelectorOutbounds := make([]string, 0, len(opts.Outbounds[i].SelectorOptions.Outbounds)+1)
				newSelectorOutbounds = append(newSelectorOutbounds, convertedOutbound.Tag)
				newSelectorOutbounds = append(newSelectorOutbounds, opts.Outbounds[i].SelectorOptions.Outbounds...)
				opts.Outbounds[i].SelectorOptions.Outbounds = newSelectorOutbounds
			case "urltest":
				newURLTestOutbounds := make([]string, 0, len(opts.Outbounds[i].URLTestOptions.Outbounds)+1)
				newURLTestOutbounds = append(newURLTestOutbounds, convertedOutbound.Tag)
				newURLTestOutbounds = append(newURLTestOutbounds, opts.Outbounds[i].URLTestOptions.Outbounds...)
				opts.Outbounds[i].URLTestOptions.Outbounds = newURLTestOutbounds
			}
		}
	}

	filtered := opts.Outbounds[:0]
	for _, outbound := range opts.Outbounds {
		if outbound.Tag != "test" {
			filtered = append(filtered, outbound)
		}
	}
	opts.Outbounds = filtered

	for i := range opts.Outbounds {
		if opts.Outbounds[i].Type == "selector" {
			opts.Outbounds[i].SelectorOptions.Outbounds = filterTags(opts.Outbounds[i].SelectorOptions.Outbounds)
		} else if opts.Outbounds[i].Type == "urltest" {
			opts.Outbounds[i].URLTestOptions.Outbounds = filterTags(opts.Outbounds[i].URLTestOptions.Outbounds)
		}
	}

	data, err := json.MarshalIndent(opts, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func filterTags(tags []string) []string {
	filtered := tags[:0]
	for _, tag := range tags {
		if tag != "test" {
			filtered = append(filtered, tag)
		}
	}
	return filtered
}
