// internal/gui/components.go
package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type Components struct {
	IPInput                   *widget.Entry
	PasswordInput             *widget.Entry
	StokInput                 *widget.Entry
	SSHPasswordInput          *widget.Entry
	SSHEnabled                *widget.Entry
	VlanIdEntry               *widget.Entry
	SingboxConfigInput        *widget.Entry
	OutboundsInput            *widget.Entry
	OutboundsCheckButton      *widget.Button
	OutboundsApplyButton      *widget.Button
	SSHLoginButton            *widget.Button
	SSHEnableButton           *widget.Button
	SSHEnablePermanentButton  *widget.Button
	InstallSingBoxBtn         *widget.Button
	InstallSingBoxPermBtn     *widget.Button
	UninstallSingBoxBtn       *widget.Button
	ConfigFileBtn             *widget.Button
	ConfigInstallFileBtn      *widget.Button
	StartSingBoxBtn           *widget.Button
	StopSingBoxBtn            *widget.Button
	RestartSingBoxBtn         *widget.Button
	InstallDnsBoxBtn          *widget.Button
	InstallPermDnsBoxBtn      *widget.Button
	UninstallDnsBoxBtn        *widget.Button
	StartDnsBoxBtn            *widget.Button
	StopDnsBoxBtn             *widget.Button
	RestartDnsBoxBtn          *widget.Button
	FirewallPatchInstallBtn   *widget.Button
	FirewallPatchUninstallBtn *widget.Button
	FirewallReloadBtn         *widget.Button
	TelegramLoginBtn          *widget.Button
	VLANButton                *widget.Button
	UARTButton                *widget.Button
	RebootButton              *widget.Button
	LogText                   *widget.TextGrid
	LogScroll                 *container.Scroll
	LogContent                string
	Spacer                    *widget.Separator
	RouterImage               *canvas.Image
}

func NewComponents() *Components {
	logText := widget.NewTextGrid()
	logScroll := container.NewScroll(logText)
	logScroll.Direction = container.ScrollBoth

	c := &Components{
		IPInput:                   widget.NewEntry(),
		PasswordInput:             widget.NewPasswordEntry(),
		StokInput:                 widget.NewEntry(),
		SSHPasswordInput:          widget.NewEntry(),
		SSHEnabled:                widget.NewEntry(),
		VlanIdEntry:               widget.NewEntry(),
		SingboxConfigInput:        widget.NewEntry(),
		OutboundsInput:            widget.NewEntry(),
		SSHLoginButton:            widget.NewButton("Get STOK", nil),
		SSHEnableButton:           widget.NewButton("Enable SSH", nil),
		SSHEnablePermanentButton:  widget.NewButton("Enable SSH permanently", nil),
		InstallSingBoxBtn:         widget.NewButton("Install Sing-box", nil),
		InstallSingBoxPermBtn:     widget.NewButton("Install Sing-box permanently", nil),
		UninstallSingBoxBtn:       widget.NewButton("Uninstall Sing-box", nil),
		ConfigFileBtn:             widget.NewButton("      Choose Sing-box config      ", nil),
		ConfigInstallFileBtn:      widget.NewButton("    Install Sing-box config file    ", nil),
		OutboundsCheckButton:      widget.NewButton("   Check Outbounds correctly   ", nil),
		OutboundsApplyButton:      widget.NewButton("    Apply Outbounds to config   ", nil),
		StartSingBoxBtn:           widget.NewButton("Start SingBox", nil),
		StopSingBoxBtn:            widget.NewButton("Stop SingBox", nil),
		RestartSingBoxBtn:         widget.NewButton("Restart SingBox", nil),
		InstallDnsBoxBtn:          widget.NewButton("Install Dns-box", nil),
		InstallPermDnsBoxBtn:      widget.NewButton("Install Dns-box permanently", nil),
		UninstallDnsBoxBtn:        widget.NewButton("Uninstall Dns-box", nil),
		StartDnsBoxBtn:            widget.NewButton("Start DnsBox", nil),
		StopDnsBoxBtn:             widget.NewButton("Stop DnsBox", nil),
		RestartDnsBoxBtn:          widget.NewButton("Restart DnsBox", nil),
		VLANButton:                widget.NewButton("Set VLAN on port 4", nil),
		UARTButton:                widget.NewButton("Set UART on", nil),
		RebootButton:              widget.NewButton("Reboot", nil),
		FirewallPatchInstallBtn:   widget.NewButton("Install Firewall Patch", nil),
		FirewallPatchUninstallBtn: widget.NewButton("Uninstall Firewall Patch", nil),
		FirewallReloadBtn:         widget.NewButton("Reload Firewall", nil),
		TelegramLoginBtn:          widget.NewButton("Telegram login", nil),
		LogText:                   logText,
		LogScroll:                 logScroll,
		Spacer:                    widget.NewSeparator(),
		RouterImage:               canvas.NewImageFromResource(nil),
	}

	c.StokInput.Disable()
	c.SSHPasswordInput.Disable()
	c.SSHEnabled.Disable()
	c.RouterImage.SetMinSize(fyne.NewSize(75, 75))
	c.RouterImage.FillMode = canvas.ImageFillContain
	c.RouterImage.Hide()
	c.OutboundsApplyButton.Disable()
	c.OutboundsCheckButton.Disable()

	c.OutboundsInput.OnChanged = func(s string) {
		if s != "" {
			c.OutboundsCheckButton.Enable()
		} else {
			c.OutboundsCheckButton.Disable()
		}
	}

	c.LogScroll.SetMinSize(fyne.NewSize(600, 400))

	return c
}

func (c *Components) BuildUI() fyne.CanvasObject {
	passwordContainer := container.NewHBox(
		container.NewVBox(
			widget.NewLabel("Password:                                                       "),
			c.PasswordInput,
		),
		layout.NewSpacer(),
		c.RouterImage,
	)

	stokBorder := container.NewBorder(nil, nil, nil,
		widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
			clip := fyne.CurrentApp().Driver().AllWindows()[0].Clipboard()
			clip.SetContent(c.StokInput.Text)
		}), c.StokInput)

	sshPassBorder := container.NewBorder(nil, nil, nil,
		widget.NewButtonWithIcon("", theme.ContentCopyIcon(), func() {
			clip := fyne.CurrentApp().Driver().AllWindows()[0].Clipboard()
			clip.SetContent(c.SSHPasswordInput.Text)
		}), c.SSHPasswordInput)

	vlanContainer := container.NewHBox(
		c.VlanIdEntry,
		c.VLANButton,
		c.UARTButton,
		c.RebootButton,
	)
	sshButtons := container.NewGridWithColumns(3,
		c.SSHLoginButton,
		c.SSHEnableButton,
		c.SSHEnablePermanentButton,
	)

	singboxInstallButtons := container.NewGridWithColumns(3,
		//c.TelegramLoginBtn,
		c.InstallSingBoxBtn,
		c.InstallSingBoxPermBtn,
		c.UninstallSingBoxBtn,
	)

	singboxControlButtons := container.NewGridWithColumns(3,
		c.StartSingBoxBtn,
		c.StopSingBoxBtn,
		c.RestartSingBoxBtn,
	)

	dnsboxInstallButtons := container.NewGridWithColumns(3,
		c.InstallDnsBoxBtn,
		c.InstallPermDnsBoxBtn,
		c.UninstallDnsBoxBtn,
	)
	dnsboxControlButtons := container.NewGridWithColumns(3,
		c.StartDnsBoxBtn,
		c.StopDnsBoxBtn,
		c.RestartDnsBoxBtn,
	)

	firewallPatchInstallBtn := container.NewGridWithColumns(3,
		c.FirewallPatchInstallBtn,
		c.FirewallPatchUninstallBtn,
		c.FirewallReloadBtn,
	)

	return container.NewVBox(
		widget.NewLabel("IP Address:"), c.IPInput,
		passwordContainer,
		widget.NewLabel("Stok (get automatic):"), stokBorder,
		widget.NewLabel("SSH Password (calculated automatic):"), sshPassBorder,
		c.Spacer,
		widget.NewLabel("SSH Actions:"), sshButtons,
		c.Spacer,
		widget.NewLabel("SingBox Install / Uninstall / Configure:"), singboxInstallButtons,
		container.NewBorder(nil, nil, c.ConfigFileBtn, c.ConfigInstallFileBtn, c.SingboxConfigInput),
		container.NewBorder(nil, nil, c.OutboundsCheckButton, c.OutboundsApplyButton, c.OutboundsInput),
		singboxControlButtons,
		c.Spacer,
		widget.NewLabel("DNSBox Control / Install:"), dnsboxInstallButtons, dnsboxControlButtons,
		c.Spacer,
		widget.NewLabel("Firewall Control / Install:"), firewallPatchInstallBtn,
		widget.NewLabel("VLAN/UART:"), vlanContainer,
		c.Spacer,
		c.LogScroll,
	)
}
