package services

import "os"

type ConfigManager struct{}

func NewConfigManager() *ConfigManager {
	return &ConfigManager{}
}

func (cm *ConfigManager) SaveFile(name string, data []byte) error {
	return os.WriteFile(name, data, 0644)
}
