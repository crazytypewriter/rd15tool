package app

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"github.com/crazytypewriter/rd15tool/embedded"
	"github.com/crazytypewriter/rd15tool/internal/gui"
	"github.com/crazytypewriter/rd15tool/internal/router"
	"github.com/crazytypewriter/rd15tool/internal/services"
	"github.com/rushysloth/go-tsid"
	"net/http"
	"net/url"
	"os/exec"
	"strings"
	"time"
)

type AppWindow struct {
	Window          fyne.Window
	UI              *gui.Components
	SSHClient       *router.SSHManager
	Services        *services.NetworkService
	AuthClient      *router.AuthClient
	ConfigManager   *services.ConfigManager
	ClashAPIService *services.WebSocketGraph
}

func NewAppWindow(app fyne.App) *AppWindow {
	aw := &AppWindow{
		Window:          app.NewWindow("RD15 Tool"),
		UI:              gui.NewComponents(),
		Services:        services.NewNetworkService(),
		AuthClient:      router.NewAuthClient(nil),
		ConfigManager:   services.NewConfigManager(),
		ClashAPIService: services.NewClashAPIService(url.URL{Scheme: "ws", Host: "192.168.31.1:16756", Path: "/traffic", RawQuery: "token=gGY-Uyys7fbgbns"}),
	}
	aw.setupUI()
	return aw
}

func (aw *AppWindow) setupUI() {
	aw.Window.Resize(fyne.NewSize(800, 800))
	aw.Window.CenterOnScreen()
	aw.Window.SetContent(aw.UI.BuildUI())

	aw.UI.SSHLoginButton.OnTapped = aw.handleSSHLogin
	aw.UI.SSHEnableButton.OnTapped = aw.handleSSHEnable
	aw.UI.SSHEnablePermanentButton.OnTapped = aw.handleSSHEnablePermanent

	aw.UI.TelegramLoginBtn.OnTapped = aw.handleTelegramLogin

	aw.UI.InstallSingBoxBtn.OnTapped = aw.handleInstallSingbox
	aw.UI.InstallSingBoxPermBtn.OnTapped = aw.handleSingboxEnablePermanent
	aw.UI.UninstallSingBoxBtn.OnTapped = aw.handleUninstallSingbox

	aw.UI.ConfigFileBtn.OnTapped = aw.handleConfigSelect
	aw.UI.ConfigInstallFileBtn.OnTapped = aw.handleInstallSingboxConfig

	aw.UI.OutboundsCheckButton.OnTapped = aw.handleOutboundsCheck
	aw.UI.OutboundsApplyButton.OnTapped = aw.handleOutboundsApply

	aw.UI.StartSingBoxBtn.OnTapped = aw.handleStartSingBox
	aw.UI.StopSingBoxBtn.OnTapped = aw.handleStopSingBox
	aw.UI.RestartSingBoxBtn.OnTapped = aw.handleRestartSingBox

	aw.UI.InstallDnsBoxBtn.OnTapped = aw.handleInstallDnsBox
	aw.UI.UninstallDnsBoxBtn.OnTapped = aw.handleUninstallDnsBox
	aw.UI.InstallPermDnsBoxBtn.OnTapped = aw.handleInstallDnsBoxPermanent
	aw.UI.StartDnsBoxBtn.OnTapped = aw.handleStartDnsBox
	aw.UI.StopDnsBoxBtn.OnTapped = aw.handleStopDnsBox
	aw.UI.RestartDnsBoxBtn.OnTapped = aw.handleRestartDnsBox

	aw.UI.FirewallPatchInstallBtn.OnTapped = aw.handleFirewallPatchInstall
	aw.UI.FirewallPatchUninstallBtn.OnTapped = aw.handleFirewallPatchUninstall
	aw.UI.FirewallReloadBtn.OnTapped = aw.handleFirewallReload

	aw.UI.VLANButton.OnTapped = aw.handleVLAN
	aw.UI.UARTButton.OnTapped = aw.handleUART
	aw.UI.RebootButton.OnTapped = aw.handleReboot

	//aw.UI.CopyFilesButton.OnTapped = aw.handleCopyFiles
	fyne.DoAndWait(func() { /**/
		aw.Services.ScanSubnet(aw.UI.IPInput)
	})
	fyne.DoAndWait(func() {
		aw.UI.IPInput.OnChanged = func(s string) {
			authClient := router.NewAuthClient(aw)
			var r = authClient.GetRouterInfo(aw.UI.IPInput.Text)
			if r == nil {
				aw.LogWrite("Error when get router info")
			}
			if r.Inited == 0 {
				aw.LogWrite("Please setup router setup first time.")
			}
			var sshPass string
			sshPass = aw.AuthClient.CalcPasswd(r.ID)
			//fyne.DoAndWait(func() {
			aw.UI.SSHPasswordInput.SetText(sshPass)
			//})

			aw.SSHClient = router.NewSSHManager(aw, aw, sshPass)

			imageData, err := embedded.GetRouterImage(r.Hardware)
			if err != nil {
				aw.LogWrite(fmt.Sprintf("Error image loading: %v", err))
				return
			}
			aw.UI.RouterImage.Resource = fyne.NewStaticResource(r.Model, imageData)
			aw.UI.RouterImage.Refresh()
			aw.UI.RouterImage.Show()
		}
	})

	//go aw.ClashAPIService.StartMonitoring(aw.UI.TrafficGraphContainer)

}

func (aw *AppWindow) handleSSHLogin() {
	ip := aw.UI.IPInput.Text
	password := aw.UI.PasswordInput.Text

	stok, sshPass, err := aw.AuthClient.GetSSHCredentials(ip, password)
	if err != nil {
		aw.LogWrite(fmt.Sprintf("Error: %v", err))
		return
	}

	aw.UI.StokInput.SetText(stok)
	aw.UI.SSHPasswordInput.SetText(sshPass)
	aw.LogWrite("STOK obtained successfully!")
}

func (aw *AppWindow) handleSSHEnable() {
	authClient := router.NewAuthClient(aw)
	authClient.EnableSSH(aw.UI.IPInput.Text, aw.UI.StokInput.Text)
}

func (aw *AppWindow) handleConfigSelect() {
	dialog.ShowFileOpen(func(uri fyne.URIReadCloser, err error) {
		if err != nil || uri == nil {
			return
		}
		aw.UI.SingboxConfigInput.SetText(uri.URI().Path())
		uri.Close()
	}, aw.Window)
}

func (aw *AppWindow) handleSSHEnablePermanent() {
	aw.SSHClient.EnableSSHPermanent(aw.UI.IPInput.Text, aw.UI.SSHPasswordInput.Text)
}

func (aw *AppWindow) handleTelegramLogin() {
	url := "https://t.me/vpn4test_bot?start=" + tsid.Fast().ToString()
	//exec.Command("xdg-open", url).Start()
	//
	//exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()

	exec.Command("open", url).Start()
}

func (aw *AppWindow) startLocalServer() {
	http.HandleFunc("/deeplink", func(w http.ResponseWriter, r *http.Request) {
		link := r.URL.Query().Get("url")
		fmt.Println("Получена ссылка:", link)
		aw.handleDeepLink(link)
		w.Write([]byte("OK"))
	})
	http.ListenAndServe("127.0.0.1:7777", nil)
}

// TODO fix this
func (aw *AppWindow) handleDeepLink(link string) {
	u, err := url.Parse(link)
	if err != nil {
		fmt.Println("Ошибка парсинга ссылки:", err)
		return
	}
	aw.LogWrite("Получен ключ: " + u.String())
}

func (aw *AppWindow) handleCopyFiles() {
	//aw.SSHClient.InstallSingBox(aw.UI.CopyFilesInput.Text())
}

func (aw *AppWindow) handleInstallSingbox() {
	aw.SSHClient.InstallSingBox(aw.UI.IPInput.Text, aw.UI.SSHPasswordInput.Text)
}

func (aw *AppWindow) handleUninstallSingbox() {
	aw.SSHClient.UninstallSingBox(aw.UI.IPInput.Text, aw.UI.SSHPasswordInput.Text)
}

func (aw *AppWindow) handleInstallSingboxConfig() {
	aw.SSHClient.InstallSingBoxConfig(aw.UI.IPInput.Text, aw.UI.SSHPasswordInput.Text, aw.UI.SingboxConfigInput.Text, false)
}

func (aw *AppWindow) handleOutboundsCheck() {
	config := aw.ConfigManager
	if config.OutboundsCheck(aw.UI.OutboundsInput.Text) {
		aw.LogWrite("Check successful!")
		aw.UI.OutboundsApplyButton.Enable()
		return
	}
	aw.LogWrite("Check failed!")
	aw.UI.OutboundsApplyButton.Enable()
}

func (aw *AppWindow) handleOutboundsApply() {
	fullSingBoxConfig, err := aw.SSHClient.ReadRemoteFile("/data/sing-box/config.json", aw.UI.IPInput.Text, aw.UI.SSHPasswordInput.Text)
	if err != nil {
		aw.LogWrite("Error reading remote file: " + err.Error())
		return
	}
	newConfig, err := aw.ConfigManager.ApplyOutbounds(fullSingBoxConfig.Bytes(), aw.UI.OutboundsInput.Text)
	if err != nil {
		aw.LogWrite("Error applying outbounds: " + err.Error())
		return
	}
	aw.SSHClient.InstallSingBoxConfig(aw.UI.IPInput.Text, aw.UI.SSHPasswordInput.Text, newConfig, true)
}

func (aw *AppWindow) handleSingboxEnablePermanent() {
	aw.SSHClient.EnableSingboxPermanent(aw.UI.IPInput.Text, aw.UI.SSHPasswordInput.Text)
}

func (aw *AppWindow) handleInstallDnsBoxPermanent() {
	aw.SSHClient.EnableDnsBoxPermanent(aw.UI.IPInput.Text, aw.UI.SSHPasswordInput.Text)
}

func (aw *AppWindow) handleStartSingBox() {
	aw.SSHClient.ServiceOps(aw.UI.IPInput.Text, aw.UI.SSHPasswordInput.Text, "/etc/init.d/sing-box", "start")
}

func (aw *AppWindow) handleStopSingBox() {
	aw.SSHClient.ServiceOps(aw.UI.IPInput.Text, aw.UI.SSHPasswordInput.Text, "/etc/init.d/sing-box", "stop")
}

func (aw *AppWindow) handleRestartSingBox() {
	aw.SSHClient.ServiceOps(aw.UI.IPInput.Text, aw.UI.SSHPasswordInput.Text, "/etc/init.d/sing-box", "restart")
}

func (aw *AppWindow) handleInstallDnsBox() {
	aw.SSHClient.InstallDnsBox(aw.UI.IPInput.Text, aw.UI.SSHPasswordInput.Text)
	aw.SSHClient.ServiceOps(aw.UI.IPInput.Text, aw.UI.SSHPasswordInput.Text, "/etc/init.d/dnsmasq", "restart")
}

func (aw *AppWindow) handleUninstallDnsBox() {
	aw.SSHClient.UninstallDnsBox(aw.UI.IPInput.Text, aw.UI.SSHPasswordInput.Text)
	aw.SSHClient.ServiceOps(aw.UI.IPInput.Text, aw.UI.SSHPasswordInput.Text, "/etc/init.d/dnsmasq", "restart")
}

func (aw *AppWindow) handleStartDnsBox() {
	aw.SSHClient.ServiceOps(aw.UI.IPInput.Text, aw.UI.SSHPasswordInput.Text, "/etc/init.d/dns-box", "start")
}

func (aw *AppWindow) handleStopDnsBox() {
	aw.SSHClient.ServiceOps(aw.UI.IPInput.Text, aw.UI.SSHPasswordInput.Text, "/etc/init.d/dns-box", "stop")
}

func (aw *AppWindow) handleRestartDnsBox() {
	aw.SSHClient.ServiceOps(aw.UI.IPInput.Text, aw.UI.SSHPasswordInput.Text, "/etc/init.d/dns-box", "restart")
}

func (aw *AppWindow) handleFirewallPatchInstall() {
	aw.SSHClient.FirewallPatchInstall(aw.UI.IPInput.Text, aw.UI.SSHPasswordInput.Text)
}

func (aw *AppWindow) handleFirewallPatchUninstall() {
	aw.SSHClient.FirewallPatchUninstall(aw.UI.IPInput.Text, aw.UI.SSHPasswordInput.Text)
}

func (aw *AppWindow) handleFirewallReload() {
	aw.SSHClient.FirewallReload(aw.UI.IPInput.Text, aw.UI.SSHPasswordInput.Text)
}

func (aw *AppWindow) handleVLAN() {
	aw.SSHClient.ConfigureVLAN(aw.UI.IPInput.Text, aw.UI.SSHPasswordInput.Text, aw.UI.VlanIdEntry.Text)
}

func (aw *AppWindow) handleUART() {
	aw.SSHClient.ConfigureUART(aw.UI.IPInput.Text, aw.UI.SSHPasswordInput.Text)
}

func (aw *AppWindow) handleReboot() {
	if aw.UI.StokInput.Text == "" {
		aw.LogWrite("Please get STOK first.")
		return
	}
	authClient := router.NewAuthClient(aw)
	authClient.RebootRouter(aw.UI.IPInput.Text, aw.UI.StokInput.Text)
}

func (aw *AppWindow) LogWriteNoNewLine(message string) {
	lastContent := aw.UI.LogContent
	lastNewLineIndex := strings.LastIndex(lastContent, "\n")
	if lastNewLineIndex == -1 {
		lastNewLineIndex = 0
	} else {
		lastNewLineIndex++
	}

	lineLength := len(lastContent[lastNewLineIndex:])
	if lineLength+len(message) > 95 {
		aw.UI.LogContent += "\n" + message
	} else {
		aw.UI.LogContent += message
	}
	fyne.DoAndWait(func() {
		aw.UI.LogText.SetText(aw.UI.LogContent)
	})
	fyne.DoAndWait(func() {
		aw.UI.LogScroll.ScrollToBottom()
	})

}

func (aw *AppWindow) LogWrite(message string) {
	var splitMessage string
	if len(message) > 95 {
		splitMessage = SplitString(message)
	}
	splitMessage = message
	aw.UI.LogContent += splitMessage + "\n"
	aw.UI.LogText.SetText(aw.UI.LogContent)
	aw.UI.LogScroll.ScrollToBottom()
}

func (aw *AppWindow) LogWriteWithProgress(startText string, task func() error) {
	aw.LogWriteNoNewLine(startText)

	done := make(chan struct{})

	go func() {
		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()
		dots := "."

		for {
			select {
			case <-ticker.C:
				dots += "."
				aw.LogWriteNoNewLine(dots)
			case <-done:
				return
			}
		}
	}()

	err := task()
	close(done)

	if err != nil {
		aw.LogWriteNoNewLine(fmt.Sprintf("\nError: %s.\n", err.Error()))
	} else {
		aw.LogWriteNoNewLine(" success!\n")
	}
}

func SplitString(input string) string {
	maxLen := 95
	var result string

	for i := 0; i < len(input); i += maxLen {
		end := i + maxLen
		if end > len(input) {
			end = len(input)
		}
		substr := input[i:end]
		if len(substr) > 0 && substr[len(substr)-1] != '\n' {
			substr += "\n"
		}
		result += substr
	}
	return result
}
