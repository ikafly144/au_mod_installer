package launcher

import (
	"au_mod_installer/pkg/aumgr"
	"au_mod_installer/ui/common"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type Launcher struct {
	state           *common.State
	launchButton    *widget.Button
	greetingContent *widget.Label

	canLaunchListener binding.DataListener
}

var _ common.Tab = (*Launcher)(nil)

func NewLauncherTab(s *common.State) common.Tab {
	var l Launcher
	l = Launcher{
		state:           s,
		launchButton:    widget.NewButtonWithIcon(lang.LocalizeKey("launcher.launch", "起動"), theme.MediaPlayIcon(), l.runLaunch),
		greetingContent: widget.NewLabelWithStyle("現在開発中！", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
	}

	l.init()

	return &l
}

func (l *Launcher) init() {
	if l.canLaunchListener == nil {
		l.canLaunchListener = binding.NewDataListener(l.checkLaunchState)
		l.state.CanLaunch.AddListener(l.canLaunchListener)
		l.checkLaunchState()
	}
	l.greetingContent.Wrapping = fyne.TextWrapWord
	l.greetingContent.Importance = widget.HighImportance
}

func (l *Launcher) Tab() (*container.TabItem, error) {
	content := container.New(
		layout.NewBorderLayout(nil, l.launchButton, nil, nil),
		container.NewVBox(
			widget.NewRichTextFromMarkdown("## "+lang.LocalizeKey("installer.select_install_path", "Among Usのインストール先を選択")),
			l.state.InstallSelect,
			widget.NewSeparator(),
			widget.NewCard("Among Us Mod Launcher", "バージョン："+l.state.Version, l.greetingContent),
		),
		l.launchButton,
	)
	return container.NewTabItem(lang.LocalizeKey("launcher.tab_name", "ランチャー"), content), nil
}

func (l *Launcher) runLaunch() {
	l.state.ErrorText.Hide()
	path, err := l.state.SelectedGamePath.Get()
	if err != nil || path == "" {
		l.state.ErrorText.Segments = []widget.RichTextSegment{
			&widget.TextSegment{Text: lang.LocalizeKey("launcher.error.no_path", "ゲームパスが指定されていません。"), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
		}
		l.state.ErrorText.Refresh()
		l.state.ErrorText.Show()
		return
	}

	if err := aumgr.LaunchAmongUs(aumgr.DetectLauncherType(path), path); err != nil {
		l.state.ErrorText.Segments = []widget.RichTextSegment{
			&widget.TextSegment{Text: "Among Usの起動に失敗しました: " + err.Error(), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
		}
		l.state.ErrorText.Refresh()
		l.state.ErrorText.Show()
		slog.Warn("Failed to launch Among Us", "error", err)
		return
	}
}

func (l *Launcher) checkLaunchState() {
	canLaunch, err := l.state.CanLaunch.Get()
	if err != nil {
		slog.Error("Failed to get CanLaunch state", "error", err)
		return
	}
	if canLaunch {
		l.launchButton.Enable()
	} else {
		l.launchButton.Disable()
	}
}
