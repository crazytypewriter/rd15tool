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
	deviceTabContent          fyne.CanvasObject
	singboxTabContent         fyne.CanvasObject
	dnsboxTabContent          fyne.CanvasObject
	firewallTabContent        fyne.CanvasObject
	vlanTabContent            fyne.CanvasObject // New tab content
	logTabContent             fyne.CanvasObject
	tabbedContainer           *container.AppTabs

	TrafficGraphContainer *fyne.Container
	//CopyFilesInput        *widget.Entry
	//CopyFilesButton       *widget.Button
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
		TrafficGraphContainer:     container.NewWithoutLayout(),
	}

	c.StokInput.Disable()
	//c.SSHPasswordInput.Disable()
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

	c.deviceTabContent = c.buildDeviceTab()
	c.singboxTabContent = c.buildSingBoxTab()
	c.dnsboxTabContent = c.buildDnsBoxTab()
	c.firewallTabContent = c.buildFirewallTab()
	c.vlanTabContent = c.buildVLANTab() // Build VLAN tab content
	return c
}

func (c *Components) buildDeviceTab() fyne.CanvasObject {
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

	sshButtons := container.NewGridWithColumns(3,
		c.SSHLoginButton,
		c.SSHEnableButton,
		c.SSHEnablePermanentButton,
	)

	sshLog := container.New(layout.NewVBoxLayout(),
		c.LogScroll,
	)

	return container.NewVBox(
		widget.NewLabel("IP Address:"), c.IPInput,
		passwordContainer,
		widget.NewLabel("Stok (get automatic):"), stokBorder,
		widget.NewLabel("SSH Password (calculated automatic):"), sshPassBorder,
		c.Spacer,
		widget.NewLabel("SSH Actions:"), sshButtons,
		// VLAN/UART and Reboot buttons are moved to the VLAN tab
		sshLog,
	)
}

func (c *Components) buildSingBoxTab() fyne.CanvasObject {
	singboxChart := container.NewVBox(
		c.TrafficGraphContainer,
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

	singboxLog := container.New(layout.NewVBoxLayout(),
		layout.NewSpacer(),
		layout.NewSpacer(),
		layout.NewSpacer(),
		c.LogScroll,
	)

	return container.NewVBox(

		widget.NewLabel("SingBox Install / Uninstall / Configure:"), singboxInstallButtons,
		container.NewBorder(nil, nil, c.ConfigFileBtn, c.ConfigInstallFileBtn, c.SingboxConfigInput),
		container.NewBorder(nil, nil, c.OutboundsCheckButton, c.OutboundsApplyButton, c.OutboundsInput),
		singboxControlButtons,
		layout.NewSpacer(),
		singboxChart,
		layout.NewSpacer(),
		layout.NewSpacer(),
		layout.NewSpacer(),
		layout.NewSpacer(),
		layout.NewSpacer(),

		singboxLog,
	)
}

func (c *Components) buildDnsBoxTab() fyne.CanvasObject {
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

	dnsboxLog := container.New(layout.NewVBoxLayout(),
		c.LogScroll,
	)

	return container.NewVBox(
		widget.NewLabel("DNSBox Control / Install:"), dnsboxInstallButtons, dnsboxControlButtons,
		dnsboxLog,
	)
}

func (c *Components) buildFirewallTab() fyne.CanvasObject {
	firewallPatchInstallBtn := container.NewGridWithColumns(3,
		c.FirewallPatchInstallBtn,
		c.FirewallPatchUninstallBtn,
		c.FirewallReloadBtn,
	)

	firewallLog := container.New(layout.NewVBoxLayout(),
		c.LogScroll,
	)

	return container.NewVBox(
		widget.NewLabel("Firewall Control / Install:"), firewallPatchInstallBtn,
		firewallLog,
	)
}

func (c *Components) buildVLANTab() fyne.CanvasObject {
	vlanContainer := container.NewHBox(
		widget.NewLabel("VLAN ID:"),
		c.VlanIdEntry,
		c.VLANButton,
	)
	uartRebootContainer := container.NewHBox(
		c.UARTButton,
		c.RebootButton,
		layout.NewSpacer(), // Add spacer to push buttons to the left if needed
	)

	uartRebootLog := container.New(layout.NewVBoxLayout(),
		c.LogScroll,
	)

	return container.NewVBox(
		widget.NewLabel("VLAN Configuration:"),
		vlanContainer,
		widget.NewLabel("UART / Reboot:"),
		uartRebootContainer,
		uartRebootLog,
	)
}

func (c *Components) BuildUI() fyne.CanvasObject {
	c.tabbedContainer = container.NewAppTabs(
		container.NewTabItem("Device  ", c.deviceTabContent),
		container.NewTabItem("VLAN", c.vlanTabContent),
		container.NewTabItem("Sing-box", c.singboxTabContent),
		container.NewTabItem("DNSBox", c.dnsboxTabContent),
		container.NewTabItem("Firewall", c.firewallTabContent),
	)

	c.tabbedContainer.SetTabLocation(container.TabLocationLeading)

	return c.tabbedContainer
}
