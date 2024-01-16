package main

import (
	_ "embed"
	"fyne.io/fyne/v2"
)

//go:embed NotoSansMonoCJKsc-Regular.otf
var monoFontRegular []byte

var resourceNotoSansMonoCJKscRegular = &fyne.StaticResource{
	StaticName:    "NotoSansMonoCJKsc-Regular.otf",
	StaticContent: monoFontRegular,
}
