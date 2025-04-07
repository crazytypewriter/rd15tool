package app

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type MobileTheme struct{}

var _ fyne.Theme = (*MobileTheme)(nil)

func (m MobileTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 0x1A, G: 0x1B, B: 0x26, A: 0xFF} // Темный фон
	case theme.ColorNameForeground:
		return color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF} // Белый текст
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0xFF}
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (m MobileTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (m MobileTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m MobileTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 6 // Было 16
	case theme.SizeNameText:
		return 12 // (было 14)
	case theme.SizeNameInlineIcon:
		return 12 // Было 20
	case theme.SizeNameSeparatorThickness: // Разделители
		return 0 // Было 2
	case theme.SizeNameScrollBar: // Полоса прокрутки
		return 8 // Было 12
	default:
		return theme.DefaultTheme().Size(name) * 0.8
	}
}
