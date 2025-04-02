// internal/router/ssh.go
package router

import (
	"bytes"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"io.rd15.tool/embedded"
	"io.rd15.tool/pkg/interfaces"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

type SSHManager struct {
	Client                *ssh.Client
	logWriter             interfaces.LogWriter
	logWriterWithProgress interfaces.LogWriterWithProgress
}

func NewSSHManager(logWriter interfaces.LogWriter, logWriterWithProgress interfaces.LogWriterWithProgress) *SSHManager {
	return &SSHManager{
		logWriter:             logWriter,
		logWriterWithProgress: logWriterWithProgress,
	}
}

func (sm *SSHManager) Connect(ip, sshPass string) (*ssh.Client, error) {
	config := &ssh.ClientConfig{
		User:            "root",
		Auth:            []ssh.AuthMethod{ssh.Password(sshPass)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client, err := ssh.Dial("tcp", ip+":22", config)
	if err != nil {
		return nil, err
	}
	sm.Client = client
	return client, nil
}

type Response struct {
	Code int `json:"code"`
}

func (sm *SSHManager) EnableSSHPermanent(ip, password string) bool {
	client, err := sm.Connect(ip, password)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Failed to ssh connect: %v", err))
		return false
	}
	defer client.Close()

	if !sm.copyEmbeddedFileWithProgress(client, "Copying sing-box patch", embedded.SshPatch, "/etc/crontabs/patches/ssh_patch.sh") {
		return false
	}
	sm.logWriter.LogWrite("SSH patch installed to disk!")

	cmdR := "crontab -l > /tmp/current_crontab && if ! grep -q 'ssh_patch.sh' /tmp/current_crontab; then echo '*/1 * * * * /etc/crontabs/patches/ssh_patch.sh >/dev/null 2>&1' >> /tmp/current_crontab; crontab /tmp/current_crontab; fi"
	_, err = runSSHCommand(client, cmdR)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Failed to add SSH check to cron: %v", err))
		return false
	}
	sm.logWriter.LogWrite("SSH installed!")

	err = sm.restartCron(client)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Failed to restart cron: %v", err))
		return false
	}
	sm.logWriter.LogWrite("SSH login and script copied successfully.")
	defer client.Close()
	return true
}

func (sm *SSHManager) EnableSingboxPermanent(ip, password string) bool {
	client, err := sm.Connect(ip, password)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Failed to ssh connect: %v", err))
		return false
	}
	defer client.Close()

	if !sm.copyEmbeddedFileWithProgress(client, "Copying sing-box patch", embedded.SingBoxPatch, "/etc/crontabs/patches/singbox_patch.sh") {
		return false
	}
	sm.logWriter.LogWrite(fmt.Sprintf("Sing-box patch installed to disk!"))

	cmdR := "crontab -l > /tmp/current_crontab && if ! grep -q 'singbox_patch.sh' /tmp/current_crontab; then echo '*/1 * * * * /etc/crontabs/patches/singbox_patch.sh >/dev/null 2>&1' >> /tmp/current_crontab; crontab /tmp/current_crontab; fi"
	_, err = runSSHCommand(client, cmdR)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Failed to add singbox check to cron: %v.", err))
		return false
	}
	sm.logWriter.LogWrite(fmt.Sprintf("Sing-box cron task installed!"))

	err = sm.restartCron(client)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Failed to restart cron: %v", err))
		return false
	}

	sm.logWriter.LogWrite(fmt.Sprintf("Sing-box patch installed successfully."))
	return true
}

func (sm *SSHManager) InstallSingBox(ip, password string) bool {
	client, err := sm.Connect(ip, password)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Error ssh login to router %s.", err.Error()))
		return false
	}
	defer client.Close()

	if !sm.copyEmbeddedFileWithProgress(client, "Copying sing-box binary", embedded.SingBoxBinary, "/data/sing-box/sing-box") {
		return false
	}
	if !sm.copyEmbeddedFileWithProgress(client, "Copying init.d file", embedded.SingBoxIni, "/etc/init.d/sing-box") {
		return false
	}
	if !sm.copyEmbeddedFileWithProgress(client, "Copying sing-box config", embedded.SingBoxConfig, "/data/sing-box/config.json") {
		return false
	}

	return true
}

func (sm *SSHManager) UninstallSingBox(ip, password string) bool {
	client, err := sm.Connect(ip, password)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Error ssh login %s.", err.Error()))
		return false
	}
	defer client.Close()

	removeFilesCmd := "rm -rf /data/sing-box /data/etc/sing-box /etc/init.d/sing-box /etc/crontabs/patches/singbox_patch.sh"
	_, err = runSSHCommand(client, removeFilesCmd)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Error removing sing-box files: %s", err.Error()))
		return false
	}

	cmdRemove := "crontab -l > /tmp/current_crontab && sed -i '/singbox_patch.sh/d' /tmp/current_crontab && crontab /tmp/current_crontab && rm /tmp/current_crontab"
	_, err = runSSHCommand(client, cmdRemove)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Error uninstall cron task for sing-box %s.", err.Error()))
		return false
	}

	err = sm.restartCron(client)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Failed to restart cron: %v", err))
		return false
	}

	sm.logWriter.LogWrite(fmt.Sprintf("Sing-box uninstall success!"))
	return true
}

func (sm *SSHManager) InstallSingBoxConfig(ip, password, config string) bool {
	client, err := sm.Connect(ip, password)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Error ssh login %s.", err.Error()))
		return false
	}
	defer client.Close()

	if strings.HasPrefix(config, "file://") {
		config = strings.TrimPrefix(config, "file://")
	}
	err = copyFileToRemote(client, config, "/data/sing-box/config.json")
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Error copying config file to router %s.", err.Error()))
		return false
	}
	sm.logWriter.LogWrite(fmt.Sprintf("Sing-box config file copied to router success!."))
	return true
}

func (sm *SSHManager) ServiceOps(ip, password, service, command string) bool {
	client, err := sm.Connect(ip, password)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Error ssh login %s.", err.Error()))
		return false
	}
	defer client.Close()

	_, err = runSSHCommand(client, service, command)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Error %v, when sending command: %s, service %s", err.Error(), command, service))
		return false
	}
	sm.logWriter.LogWrite(fmt.Sprintf("Service %s %s successful!.", service, command))
	return true
}

func (sm *SSHManager) InstallDnsBox(ip, password string) bool {
	client, err := sm.Connect(ip, password)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Error ssh login to router %s.", err.Error()))
		return false
	}
	defer client.Close()

	if !sm.copyEmbeddedFileWithProgress(client, "Copying dns-box binary", embedded.DnsBoxBinary, "/data/dns-box/dns-box") {
		return false
	}
	if !sm.copyEmbeddedFileWithProgress(client, "Copying dns-box init.d file", embedded.DnsBoxIni, "/etc/init.d/dns-box") {
		return false
	}
	if !sm.copyEmbeddedFileWithProgress(client, "Copying dns-box config", embedded.DnsBoxConfig, "/data/dns-box/config.json") {
		return false
	}

	if !sm.ChangeDnsMasqConfig(client, true) {
		return false
	}

	return true
}

func (sm *SSHManager) ChangeDnsMasqConfig(client *ssh.Client, add bool) bool {
	fileModifications := map[string]map[string]string{}

	if add {
		// Добавляем нужные строки
		fileModifications["/etc/config/dhcp"] = map[string]string{
			`(?m)(option resolvfile '\/tmp\/resolv\.conf\.d\/resolv\.conf\.auto')`: "$1\n        option noresolv '1'",
			`(?m)(config dnsmasq)`: "$1\n        list server '127.0.0.1#953'",
		}
	} else {
		// Удаляем строки
		fileModifications["/etc/config/dhcp"] = map[string]string{
			`(?m)^\s*option noresolv '1'\n?`:            "",
			`(?m)^\s*list server '127\.0\.0\.1#953'\n?`: "",
		}
	}

	for filePath, patterns := range fileModifications {
		replacements := make(map[*regexp.Regexp]string)
		for pattern, replacement := range patterns {
			re, err := regexp.Compile(pattern)
			if err != nil {
				fmt.Println("Error compiling regex:", err)
				sm.logWriter.LogWrite(fmt.Sprintf("Error compiling regex: %s.\n", err.Error()))
				return false
			}
			replacements[re] = replacement
		}

		if err := sm.replaceRemoteFileRegex(client, filePath, replacements); err != nil {
			sm.logWriter.LogWrite(fmt.Sprintf("Error updating: %s", err))
			return false
		}
	}

	return true
}

func (sm *SSHManager) UninstallDnsBox(ip, password string) bool {
	client, err := sm.Connect(ip, password)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Error ssh login %s.", err.Error()))
		return false
	}
	defer client.Close()

	removeFilesCmd := "rm -rf /data/dns-box /data/etc/dns-box /etc/init.d/dns-box /etc/crontabs/patches/dnsbox_patch.sh"
	_, err = runSSHCommand(client, removeFilesCmd)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Error removing dns-box files: %s", err.Error()))
		return false
	}

	cmdRemove := "crontab -l > /tmp/current_crontab && sed -i '/dnsbox_patch.sh/d' /tmp/current_crontab && crontab /tmp/current_crontab && rm /tmp/current_crontab"
	_, err = runSSHCommand(client, cmdRemove)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Error uninstall cron task for dns-box %s.", err.Error()))
		return false
	}

	err = sm.restartCron(client)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Failed to restart cron: %v", err))
		return false
	}

	if !sm.ChangeDnsMasqConfig(client, true) {
		return false
	}

	sm.logWriter.LogWrite(fmt.Sprintf("Sing-box uninstall success!"))
	return true
}

func (sm *SSHManager) EnableDnsBoxPermanent(ip, password string) bool {
	client, err := sm.Connect(ip, password)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Failed to ssh connect: %v", err))
		return false
	}
	defer client.Close()

	if !sm.copyEmbeddedFileWithProgress(client, "Copying dns-box patch", embedded.DnsBoxPatch, "/etc/crontabs/patches/dnsbox_patch.sh") {
		return false
	}
	sm.logWriter.LogWrite(fmt.Sprintf("Dns-box patch installed to disk!"))

	cmdR := "crontab -l > /tmp/current_crontab && if ! grep -q 'dnsbox_patch.sh' /tmp/current_crontab; then echo '*/1 * * * * /etc/crontabs/patches/dnsbox_patch.sh >/dev/null 2>&1' >> /tmp/current_crontab; crontab /tmp/current_crontab; fi"
	_, err = runSSHCommand(client, cmdR)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Failed to add dnsbox check to cron: %v.", err))
		return false
	}
	sm.logWriter.LogWrite(fmt.Sprintf("dns-box cron task installed!"))

	err = sm.restartCron(client)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Failed to restart cron: %v", err))
		return false
	}

	sm.logWriter.LogWrite(fmt.Sprintf("Sing-box patch installed successfully."))
	return true
}

func (sm *SSHManager) FirewallPatchInstall(ip, password string) bool {
	client, err := sm.Connect(ip, password)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Failed to ssh connect: %v", err))
		return false
	}
	defer client.Close()

	if !sm.copyEmbeddedFileWithProgress(client, "Copying firewall patch", embedded.FirewallPatch, "/etc/crontabs/patches/firewall_patch.sh") {
		return false
	}

	cmdR := "crontab -l > /tmp/current_crontab && if ! grep -q 'firewall_patch.sh' /tmp/current_crontab; then echo '*/1 * * * * /etc/crontabs/patches/firewall_patch.sh >/dev/null 2>&1' >> /tmp/current_crontab; crontab /tmp/current_crontab; fi"
	_, err = runSSHCommand(client, cmdR)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Failed to add firewall check to cron: %v.", err))
		return false
	}
	sm.logWriter.LogWrite(fmt.Sprintf("Firewall cron task installed!"))

	err = sm.restartCron(client)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Failed to restart cron: %v", err))
		return false
	}
	sm.logWriter.LogWrite("Firewall patch installed successfully!")
	return true
}

func (sm *SSHManager) FirewallPatchUninstall(ip, password string) bool {
	client, err := sm.Connect(ip, password)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Failed to ssh connect: %v", err))
		return false
	}
	defer client.Close()

	removeFilesCmd := "rm -rf /data/userdisk/appdata/firewall.sh /etc/crontabs/patches/firewall_patch.sh"
	_, err = runSSHCommand(client, removeFilesCmd)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Error removing dns-box files: %s", err.Error()))
		return false
	}

	cmdRemove := "crontab -l > /tmp/current_crontab && sed -i '/firewall_patch.sh/d' /tmp/current_crontab && crontab /tmp/current_crontab && rm /tmp/current_crontab"
	_, err = runSSHCommand(client, cmdRemove)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Error uninstall task for firewall:  %s.", err.Error()))
		return false
	}

	err = sm.restartCron(client)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Failed to restart cron: %v", err))
		return false
	}

	sm.logWriter.LogWrite(fmt.Sprintf("Firewall uninstall success!"))
	return true

}

func (sm *SSHManager) FirewallReload(ip, password string) bool {
	client, err := sm.Connect(ip, password)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Error ssh login %s.", err.Error()))
		return false
	}
	defer client.Close()

	cmdRun := "/data/userdisk/appdata/firewall.sh reload"
	_, err = runSSHCommand(client, cmdRun)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Error reloading firewall:  %s.", err.Error()))
		return false
	}
	return false
}
func (sm *SSHManager) copyEmbeddedFileWithProgress(client *ssh.Client, description string, data []byte, remotePath string) bool {
	var copyErr error
	task := func() error {
		reader := bytes.NewReader(data)
		copyErr = copyBinaryToRemote(client, reader, int64(len(data)), remotePath, 0755)
		return copyErr
	}
	sm.logWriterWithProgress.LogWriteWithProgress(description, task)
	return copyErr == nil
}

func (sm *SSHManager) restartCron(client *ssh.Client) error {
	_, err := runSSHCommand(client, "/etc/init.d/cron restart")
	if err != nil {
		return err
	}
	sm.logWriter.LogWrite("Cron restarted successfully!")
	return nil
}

func (sm *SSHManager) ConfigureVLAN(ip, password, vlanID string) bool {
	client, err := sm.Connect(ip, password)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Error ssh login %s.", err.Error()))
		return false
	}
	defer client.Close()

	id := vlanID
	newReplacement := "." + id
	if id == "0" || id == "" {
		newReplacement = ""
	}

	fileModifications := map[string]map[string]string{
		"/etc/config/network": {
			`config interface 'eth1(\.\d+)?'`:      "config interface 'eth1'",
			`option ifname '([^']*?)eth1(\.\d+)?'`: "option ifname '${1}eth1" + newReplacement + "'",
		},
		"/etc/config/port_map": {
			`option ifname 'eth1(\.\d+)?'`: "option ifname 'eth1" + newReplacement + "'",
		},
	}

	for filePath, patterns := range fileModifications {
		replacements := make(map[*regexp.Regexp]string)
		for pattern, replacement := range patterns {
			re, err := regexp.Compile(pattern)
			if err != nil {
				sm.logWriter.LogWrite(fmt.Sprintf("Error compiling regex: %s.", err.Error()))
				return false
			}
			replacements[re] = replacement
		}

		if err := sm.replaceRemoteFileRegex(client, filePath, replacements); err != nil {
			sm.logWriter.LogWrite(fmt.Sprintf("Error updating: %s", err.Error()))
		}
	}
	sm.logWriter.LogWrite(fmt.Sprintf("VLAN configuration updated successfully!"))
	return true
}

func (sm *SSHManager) ConfigureUART(ip, password string) bool {
	client, err := sm.Connect(ip, password)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Error ssh login %s.", err.Error()))
		return false
	}
	defer client.Close()

	fileModifications := map[string]map[string]string{
		"/etc/inittab": {
			`ttyMSM0::askfirst:/bin/ash\s+--login`: "ttyMSM0::askfirst:/bin/ash",
		},
	}

	for filePath, patterns := range fileModifications {
		replacements := make(map[*regexp.Regexp]string)
		for pattern, replacement := range patterns {
			re, err := regexp.Compile(pattern)
			if err != nil {
				fmt.Println("Error compiling regex:", err)
				sm.logWriter.LogWrite(fmt.Sprintf("Error compiling regex: %s.\n", err.Error()))
				return false
			}
			replacements[re] = replacement
		}

		if err := sm.replaceRemoteFileRegex(client, filePath, replacements); err != nil {
			sm.logWriter.LogWrite(fmt.Sprintf("Error updating: %s", err))
			return false
		}
	}
	return true
}

type SingBoxInstaller struct {
	client *ssh.Client
}

func NewSingBoxInstaller(client *ssh.Client) *SingBoxInstaller {
	return &SingBoxInstaller{client: client}
}

func (si *SingBoxInstaller) Install() error {
	// Original installSingBox logic
	return nil
}

func copyBinaryToRemote(client *ssh.Client, reader io.Reader, size int64, remotePath string, mode os.FileMode) error {
	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	defer stdin.Close()

	remoteDir := filepath.Dir(remotePath)
	remoteFileName := filepath.Base(remotePath)

	mkdirSession, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create mkdir session: %w", err)
	}
	if output, err := mkdirSession.CombinedOutput(fmt.Sprintf("mkdir -p %s", strconv.Quote(remoteDir))); err != nil {
		mkdirSession.Close()
		return fmt.Errorf("failed to create remote directory: %s: %w", string(output), err)
	}
	mkdirSession.Close()

	if err := session.Start(fmt.Sprintf("scp -t %s", remoteDir)); err != nil {
		return fmt.Errorf("failed to start remote scp command: %w", err)
	}

	header := fmt.Sprintf("C%#o %d %s\n", mode.Perm()|0111, size, remoteFileName)
	if _, err := fmt.Fprint(stdin, header); err != nil {
		return fmt.Errorf("failed to send scp header: %w", err)
	}

	if _, err := io.Copy(stdin, reader); err != nil {
		return fmt.Errorf("failed to copy binary content: %w", err)
	}

	if _, err := fmt.Fprint(stdin, "\x00"); err != nil {
		return fmt.Errorf("failed to send scp end signal: %w", err)
	}

	stdin.Close()

	if err := session.Wait(); err != nil {
		return fmt.Errorf("remote scp command failed: %w", err)
	}

	return nil
}

func runSSHCommand(client *ssh.Client, args ...string) (string, error) {
	if client == nil {
		return "", fmt.Errorf("client is nil")
	}
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	var stdoutBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = os.Stderr
	cmd := exec.Command(args[0], args[1:]...)
	fmt.Println("COMMAND TO RUN:", cmd.String())
	if err := session.Run(cmd.String()); err != nil {
		return stdoutBuf.String(), fmt.Errorf("failed to execute command: %w", err)
	}
	return stdoutBuf.String(), nil
}

func copyFileToRemote(client *ssh.Client, localPath, remotePath string) error {
	srcFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("failed to open local file: %w", err)
	}
	defer srcFile.Close()

	srcFileInfo, err := srcFile.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat local file: %w", err)
	}

	session, err := client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	pipe, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe: %w", err)
	}
	defer pipe.Close()

	cmd := fmt.Sprintf("scp -t %s", remotePath)
	if err := session.Start(cmd); err != nil {
		return fmt.Errorf("failed to start remote scp command: %w", err)
	}

	fmt.Fprintf(pipe, "C%#o %d %s\n", srcFileInfo.Mode().Perm()|0111, srcFileInfo.Size(), filepath.Base(remotePath))

	if _, err := io.Copy(pipe, srcFile); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	fmt.Fprint(pipe, "\x00")
	pipe.Close()

	if err := session.Wait(); err != nil {
		return fmt.Errorf("failed to complete scp command: %w", err)
	}

	return nil
}

func (sm *SSHManager) replaceRemoteFileRegex(client *ssh.Client, filePath string, replacements map[*regexp.Regexp]string) error {
	cmd := fmt.Sprintf("cat %s", filePath)
	fileContent, err := runSSHCommand(client, cmd)
	if err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("Error running command: %s.", err.Error()))
	}

	modified := false
	lines := strings.Split(fileContent, "\n")
	for i, line := range lines {
		for re, replacement := range replacements {
			if re.MatchString(line) {
				newLine := re.ReplaceAllString(line, replacement)
				if line != newLine {
					lines[i] = newLine
					modified = true
				}
			}
		}
	}

	if !modified {
		fmt.Println("No changes needed for", filePath)
		sm.logWriter.LogWrite(fmt.Sprintf("No changes needed for %s", filePath))
		return nil
	}

	updatedContent := strings.Join(lines, "\n")

	echoCommand := fmt.Sprintf("echo -e %q > %s", updatedContent, filePath)
	if _, err = runSSHCommand(client, echoCommand); err != nil {
		sm.logWriter.LogWrite(fmt.Sprintf("failed to update file: %w", err))
	}

	sm.logWriter.LogWrite("File updated successfully on remote host!")
	return nil
}
