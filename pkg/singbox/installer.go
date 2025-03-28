// pkg/singbox/installer.go
package singbox

import (
	"io.rd15.tool/internal/router"
)

type Installer struct {
	SSHClient *router.SSHClient
}

func NewInstaller(sshClient *router.SSHClient) *Installer {
	return &Installer{SSHClient: sshClient}
}

func (i *Installer) Install() error {
	_, err := i.SSHClient.Execute("mkdir -p /data/sing-box")
	if err != nil {
		return err
	}
	// Остальная логика установки...
	return nil
}
