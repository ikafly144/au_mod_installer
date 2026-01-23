package uicommon

import (
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

func (s *State) ShowEpicLoginWindow(onSuccess func(), onCancel func()) {
	codeEntry := widget.NewEntry()
	codeEntry.PlaceHolder = lang.LocalizeKey("settings.epic_login_code_label", "Authorization Code")

	var popup dialog.Dialog
	success := false

	loginBtn := widget.NewButton(lang.LocalizeKey("settings.epic_login", "Login"), func() {
		code := codeEntry.Text
		if code == "" {
			return
		}

		// Disable button to prevent double click?
		// But Fyne doesn't make it easy to re-enable inside closure without passing reference.

		session, err := s.Core.EpicApi.LoginWithAuthCode(code)
		if err != nil {
			dialog.ShowError(err, s.Window)
			return
		}

		if err := s.Core.EpicSessionManager.Save(session); err != nil {
			dialog.ShowError(err, s.Window)
			return
		}

		success = true
		popup.Hide()
		if onSuccess != nil {
			onSuccess()
		}
	})
	loginBtn.Importance = widget.HighImportance

	// We use the dialog's dismiss button as "Cancel", so we don't need a custom Cancel button here.
	// But for layout purposes, we might want to align Login button right.

	content := container.NewVBox(
		widget.NewLabel(lang.LocalizeKey("settings.epic_login_instruction", "Epic Gamesでログインし、表示されたページのコードを以下に入力してください。")),
		widget.NewButton(lang.LocalizeKey("settings.epic_login_url_button", "ログインページを開く"), func() {
			authUrl := s.Core.EpicApi.GetAuthUrl()
			u, _ := url.Parse(authUrl)
			_ = fyne.CurrentApp().OpenURL(u) // TODO: handle error
		}),
		codeEntry,
		container.NewHBox(layout.NewSpacer(), loginBtn),
	)

	popup = dialog.NewCustom(
		lang.LocalizeKey("settings.epic_games_account", "Epic Gamesアカウント"),
		lang.LocalizeKey("common.cancel", "キャンセル"),
		content,
		s.Window,
	)

	popup.SetOnClosed(func() {
		if !success && onCancel != nil {
			onCancel()
		}
	})

	popup.Resize(fyne.NewSize(400, 250))
	popup.Show()
}
