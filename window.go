package main

import (
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/makiuchi-d/gozxing"
	nativeDialog "github.com/sqweek/dialog"
	"golang.design/x/clipboard"
	"image/color"
)

var WMain fyne.Window
var spacer *canvas.Rectangle

func InitMainWindow() fyne.Window {
	w := App.NewWindow("EasyLPAC")
	w.Resize(fyne.Size{
		Width:  850,
		Height: 545,
	})
	w.SetMaster()
	SetFixedWindowSize(&w)

	statusBar := container.NewGridWrap(fyne.Size{
		Width:  100,
		Height: DownloadButton.MinSize().Height,
	}, StatusLabel, StatusProcessBar)

	spacer = canvas.NewRectangle(color.Transparent)
	spacer.SetMinSize(fyne.NewSize(1, 1))

	topToolBar := container.NewBorder(
		layout.NewSpacer(),
		nil,
		container.New(layout.NewHBoxLayout(), OpenLogButton, spacer, RefreshButton, spacer),
		FreeSpaceLabel,
		container.NewBorder(
			nil,
			nil,
			widget.NewLabel("Card Reader:"),
			nil,
			container.NewHBox(container.NewGridWrap(fyne.Size{
				Width:  280,
				Height: ApduDriverSelect.MinSize().Height,
			}, ApduDriverSelect), ApduDriverRefreshButton)),
	)

	profileTabContent := container.NewBorder(
		topToolBar,
		container.NewBorder(
			nil,
			nil,
			nil,
			container.NewHBox(ProfileMaskCheck, DownloadButton,
				// spacer, DiscoveryButton,
				spacer, SetNicknameButton,
				spacer, SwitchStateButton,
				spacer, DeleteProfileButton),
			statusBar),
		nil,
		nil,
		ProfileList)
	ProfileTab = container.NewTabItem("Profile", profileTabContent)

	notificationTabContent := container.NewBorder(
		topToolBar,
		container.NewBorder(
			nil,
			nil,
			nil,
			container.NewHBox(NotificationMaskCheck,
				spacer, ProcessNotificationButton,
				spacer, RemoveAllNotificationButton,
				spacer, RemoveNotificationButton),
			statusBar),
		nil,
		nil,
		NotificationList)
	NotificationTab = container.NewTabItem("Notification", notificationTabContent)

	chipInfoTabContent := container.NewBorder(
		topToolBar,
		container.NewBorder(
			nil,
			nil,
			nil,
			nil,
			statusBar),
		nil,
		nil,
		container.NewBorder(
			container.NewVBox(
				container.NewHBox(
					EidLabel, CopyEidButton, layout.NewSpacer(), EUICCManufacturerLabel),
				container.NewHBox(
					DefaultDpAddressLabel, SetDefaultSmdpButton, layout.NewSpacer(), ViewCertInfoButton),
				container.NewHBox(
					RootDsAddressLabel, layout.NewSpacer(), CopyEuiccInfo2Button)),
			nil,
			nil,
			nil,
			container.NewScroll(EuiccInfo2Entry),
		))
	ChipInfoTab = container.NewTabItem("Chip Info", chipInfoTabContent)

	settingsTabContent := container.NewVBox(
		&widget.Label{Text: "lpac debug output", TextStyle: fyne.TextStyle{Bold: true}},
		widget.NewCheck("Enable env LIBEUICC_DEBUG_HTTP", func(b bool) {
			ConfigInstance.DebugHTTP = b
		}),
		widget.NewCheck("Enable env LIBEUICC_DEBUG_APDU", func(b bool) {
			ConfigInstance.DebugAPDU = b
		}))
	SettingsTab = container.NewTabItem("Settings", settingsTabContent)

	thankstoText := widget.NewRichTextFromMarkdown(`
# Thanks to

[lpac](https://github.com/estkme-group/lpac) C-based eUICC LPA

[eUICC Manual](https://euicc-manual.septs.app) eUICC Developer Manual

[fyne](https://github.com/fyne-io/fyne) Material Design GUI toolkit`)

	aboutText := widget.NewRichTextFromMarkdown(`
# EasyLPAC

lpac GUI Frontend

[Github](https://github.com/creamlike1024/EasyLPAC) Repo `)

	aboutTabContent := container.NewBorder(
		nil,
		container.NewBorder(nil, nil,
			widget.NewLabel(fmt.Sprintf("Version: %s", Version)),
			widget.NewLabel(fmt.Sprintf("eUICC Data: %s", EUICCDataVersion))),
		nil,
		nil,
		container.NewCenter(container.NewVBox(thankstoText, aboutText)))
	AboutTab = container.NewTabItem("About", aboutTabContent)

	Tabs = container.NewAppTabs(ProfileTab, NotificationTab, ChipInfoTab, SettingsTab, AboutTab)

	w.SetContent(Tabs)

	return w
}

func InitDownloadDialog() dialog.Dialog {
	smdpEntry := &widget.Entry{PlaceHolder: "Leave it empty to use default SM-DP+"}
	matchIDEntry := &widget.Entry{PlaceHolder: "Activation code. Optional"}
	confirmCodeEntry := &widget.Entry{PlaceHolder: "Optional"}
	imeiEntry := &widget.Entry{PlaceHolder: "The IMEI sent to SM-DP+. Optional"}

	formItems := []*widget.FormItem{
		{Text: "SM-DP+", Widget: smdpEntry},
		{Text: "Matching ID", Widget: matchIDEntry},
		{Text: "Confirm Code", Widget: confirmCodeEntry},
		{Text: "IMEI", Widget: imeiEntry},
	}

	form := widget.NewForm(formItems...)
	var d dialog.Dialog
	showConfirmCodeNeededDialog := func() {
		dConfirmCodeNeeded := dialog.NewInformation("Confirm Code Required",
			"This profile needs confirm code to download.\n"+
				"Please fill the confirm code manually.", WMain)
		dConfirmCodeNeeded.Show()
	}
	cancelButton := &widget.Button{
		Text: "Cancel",
		Icon: theme.CancelIcon(),
		OnTapped: func() {
			d.Hide()
		},
	}
	downloadButton := &widget.Button{
		Text:       "Download",
		Icon:       theme.ConfirmIcon(),
		Importance: widget.HighImportance,
		OnTapped: func() {
			d.Hide()
			pullConfig := PullInfo{
				SMDP:        smdpEntry.Text,
				MatchID:     matchIDEntry.Text,
				ConfirmCode: confirmCodeEntry.Text,
				IMEI:        imeiEntry.Text,
			}
			go func() {
				RefreshNotification()
				LpacProfileDownload(pullConfig)
			}()
		},
	}
	// 回调函数需要操作这两个 Button，预先声明
	var selectQRCodeButton *widget.Button
	var pasteFromClipboardButton *widget.Button
	disableButtons := func() {
		cancelButton.Disable()
		downloadButton.Disable()
		selectQRCodeButton.Disable()
		pasteFromClipboardButton.Disable()
	}
	enableButtons := func() {
		cancelButton.Enable()
		downloadButton.Enable()
		selectQRCodeButton.Enable()
		pasteFromClipboardButton.Enable()
	}

	selectQRCodeButton = &widget.Button{
		Text: "Scan image file",
		Icon: theme.FileImageIcon(),
		OnTapped: func() {
			go func() {
				disableButtons()
				defer enableButtons()
				fileBuilder := nativeDialog.File().Title("Select a QR Code image file")
				fileBuilder.Filters = []nativeDialog.FileFilter{
					{
						Desc:       "Image (*.png, *.jpg, *.jpeg)",
						Extensions: []string{"PNG", "JPG", "JPEG"},
					},
					{
						Desc:       "All files (*.*)",
						Extensions: []string{"*"},
					},
				}

				filename, err := fileBuilder.Load()
				if err != nil {
					if err.Error() != "Cancelled" {
						panic(err)
					}
				} else {
					result, err := ScanQRCodeImageFile(filename)
					if err != nil {
						dError := dialog.NewError(err, WMain)
						dError.Show()
					} else {
						pullInfo, confirmCodeNeeded, err := DecodeLpaActivationCode(result.String())
						if err != nil {
							dError := dialog.NewError(err, WMain)
							dError.Show()
						} else {
							smdpEntry.SetText(pullInfo.SMDP)
							matchIDEntry.SetText(pullInfo.MatchID)
							if confirmCodeNeeded {
								go showConfirmCodeNeededDialog()
							}
						}
					}
				}
			}()
		},
	}
	pasteFromClipboardButton = &widget.Button{
		Text: "Paste QR Code or LPA:1 Activation Code from clipboard",
		Icon: theme.ContentPasteIcon(),
		OnTapped: func() {
			go func() {
				disableButtons()
				defer enableButtons()
				var err error
				var pullInfo PullInfo
				var confirmCodeNeeded bool
				var qrResult *gozxing.Result

				format, result, err := PasteFromClipboard()
				if err != nil {
					dError := dialog.NewError(err, WMain)
					dError.Show()
					return
				}
				switch format {
				case clipboard.FmtImage:
					qrResult, err = ScanQRCodeImageBytes(result)
					if err != nil {
						dError := dialog.NewError(err, WMain)
						dError.Show()
						return
					}
					pullInfo, confirmCodeNeeded, err = DecodeLpaActivationCode(qrResult.String())
				case clipboard.FmtText:
					pullInfo, confirmCodeNeeded, err = DecodeLpaActivationCode(CompleteActivationCode(string(result)))
				default:
					// Unreachable, should not be here.
					panic(nil)
				}
				if err != nil {
					dError := dialog.NewError(err, WMain)
					dError.Show()
					return
				}
				smdpEntry.SetText(pullInfo.SMDP)
				matchIDEntry.SetText(pullInfo.MatchID)
				if confirmCodeNeeded {
					go showConfirmCodeNeededDialog()
				}
			}()
		},
	}
	d = dialog.NewCustomWithoutButtons("Download", container.NewBorder(
		nil,
		container.NewVBox(spacer, container.NewCenter(selectQRCodeButton), spacer,
			container.NewCenter(pasteFromClipboardButton), spacer,
			container.NewCenter(container.NewHBox(cancelButton, spacer, downloadButton))),
		nil,
		nil,
		form), WMain)
	d.Resize(fyne.Size{
		Width:  520,
		Height: 380,
	})
	return d
}

func InitSetNicknameDialog() dialog.Dialog {
	entry := &widget.Entry{PlaceHolder: "Leave it empty to remove nickname"}
	form := []*widget.FormItem{
		{Text: "Nickname", Widget: entry},
	}
	d := dialog.NewForm("Set Nickname", "Submit", "Cancel", form, func(b bool) {
		if b {
			if err := LpacProfileNickname(Profiles[SelectedProfile].Iccid, entry.Text); err != nil {
				ShowLpacErrDialog(err)
			}
			RefreshProfile()
		}
	}, WMain)
	d.Resize(fyne.Size{
		Width:  400,
		Height: 180,
	})
	return d
}

func InitSetDefaultSmdpDialog() dialog.Dialog {
	entry := &widget.Entry{PlaceHolder: "Leave it empty to remove default SM-DP+ setting"}
	form := []*widget.FormItem{
		{Text: "Default SM-DP+", Widget: entry},
	}
	d := dialog.NewForm("Set Default SM-DP+", "Submit", "Cancel", form, func(b bool) {
		if b {
			if err := LpacChipDefaultSmdp(entry.Text); err != nil {
				ShowLpacErrDialog(err)
			}
			RefreshChipInfo()
		}
	}, WMain)
	d.Resize(fyne.Size{
		Width:  510,
		Height: 200,
	})
	return d
}

func ShowLpacErrDialog(err error) {
	go func() {
		l := &widget.Label{Text: fmt.Sprintf("%v", err)}
		content := container.NewVBox(
			container.NewCenter(container.NewHBox(
				widget.NewIcon(theme.ErrorIcon()),
				widget.NewLabel("lpac error"))),
			container.NewCenter(l),
			container.NewCenter(widget.NewLabel("Please check the log for details")))
		d := dialog.NewCustom("Error", "OK", content, WMain)
		d.Show()
	}()
}

func ShowSelectItemDialog() {
	go func() {
		d := dialog.NewInformation("Info", "Please select a item.", WMain)
		d.Resize(fyne.Size{
			Width:  220,
			Height: 160,
		})
		d.Show()
	}()
}

func ShowSelectCardReaderDialog() {
	go func() {
		d := dialog.NewInformation("Info", "Please select a card reader.", WMain)
		d.Show()
	}()
}

func ShowRefreshNeededDialog() {
	go func() {
		d := dialog.NewInformation("Info", "Please refresh before proceeding.\n", WMain)
		d.Show()
	}()
}
