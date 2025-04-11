package app

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
	"image/color"
)

type MacTheme struct{}

var _ fyne.Theme = (*MacTheme)(nil)

func (m MacTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 0x1E, G: 0x1E, B: 0x1E, A: 0xFF} // Насыщенный темно-серый фон
	case theme.ColorNameForeground:
		return color.NRGBA{R: 0xE0, G: 0xE0, B: 0xE0, A: 0xFF} // Светло-серый текст
	case theme.ColorNamePrimary:
		return color.NRGBA{R: 0x4A, G: 0x86, B: 0xE8, A: 0xFF} // Приглушенный синий акцент
	case theme.ColorNameHover:
		return color.NRGBA{R: 0x2E, G: 0x2E, B: 0x2E, A: 0xFF} // Еле заметное высветление при наведении
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 0x60, G: 0x60, B: 0x60, A: 0xFF}
	case theme.ColorNameInputBackground:
		return color.NRGBA{R: 0x28, G: 0x28, B: 0x28, A: 0xFF} // Более темный фон для полей ввода
	case theme.ColorNameInputBorder:
		return color.NRGBA{R: 0x40, G: 0x40, B: 0x40, A: 0xFF} // Еле заметная граница для полей ввода
	case theme.ColorNameSeparator:
		return color.NRGBA{R: 0x38, G: 0x38, B: 0x38, A: 0xFF} // Тонкие разделители
	case theme.ColorNameShadow:
		return color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x66} // Более мягкая тень
	case "LogBackground": // Кастомное имя цвета
		return color.NRGBA{R: 0x28, G: 0x28, B: 0x28, A: 0xFF}
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (m MacTheme) Font(style fyne.TextStyle) fyne.Resource {
	// Можно использовать более современный шрифт, если он у вас есть.
	// Например, "SF Pro Text" (доступен на macOS).
	// Если вы хотите использовать стандартный шрифт, оставьте эту строку.
	return theme.DefaultTheme().Font(style)
}

func (m MacTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m MacTheme) ScrollBarSize() float32 {
	return 0 // Делаем полосу прокрутки невидимой
}

func (m MacTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 8 // Немного больше отступы для "воздуха"
	case theme.SizeNameText:
		return 13 // Чуть больший размер текста для лучшей читаемости
	case theme.SizeNameInlineIcon:
		return 16 // Стандартный размер иконок
	case theme.SizeNameSeparatorThickness:
		return 1 // Тонкие разделители
	//case theme.SizeNameScrollBar:
	//	return 0
	case theme.SizeNameInputBorder:
		return 1 // Тонкая граница для полей ввода
	default:
		return theme.DefaultTheme().Size(name) * 0.9 // Небольшое уменьшение общих размеров
	}
}

type WindowsTheme struct{}

var _ fyne.Theme = (*WindowsTheme)(nil)

func (w WindowsTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return color.NRGBA{R: 0xE1, G: 0xE1, B: 0xE1, A: 0xFF} // Светлый фон для Windows
	case theme.ColorNameForeground:
		return color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xFF} // Черный текст для Windows
	case theme.ColorNameDisabled:
		return color.NRGBA{R: 0xA0, G: 0xA0, B: 0xA0, A: 0xFF}
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (w WindowsTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (w WindowsTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (w WindowsTheme) Size(name fyne.ThemeSizeName) float32 {
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
