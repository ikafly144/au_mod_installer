package installer

import (
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/ikafly144/au_mod_installer/client/ui/uicommon"
	"github.com/ikafly144/au_mod_installer/pkg/progress"
)

type Installer struct {
	state                *uicommon.State
	uninstallButton      *widget.Button
	progressBar          *progress.FyneProgress
	installationListener binding.DataListener
}

var _ uicommon.Tab = (*Installer)(nil)

func NewInstallerTab(s *uicommon.State) uicommon.Tab {
	var i Installer
	i = Installer{
		state:           s,
		uninstallButton: widget.NewButtonWithIcon(lang.LocalizeKey("installation.uninstall", "Uninstall"), theme.DeleteIcon(), i.runUninstall),
		progressBar:     progress.NewFyneProgress(widget.NewProgressBar()),
	}

	i.init()

	return &i
}

func (i *Installer) init() {
	i.uninstallButton.Importance = widget.DangerImportance
	i.uninstallButton.Disable()
	if i.installationListener == nil {
		i.installationListener = binding.NewDataListener(i.checkUninstallState)
		i.state.ModInstalled.AddListener(i.installationListener)
		i.state.SelectedGamePath.AddListener(i.installationListener)
		i.state.CanInstall.AddListener(i.installationListener)
		i.state.RefreshModInstallation()
	}
}

func (i *Installer) checkUninstallState() {
	if ok, err := i.state.CanInstall.Get(); !ok || err != nil {
		i.uninstallButton.Disable()
		return
	}
	if ok, err := i.state.ModInstalled.Get(); ok && err == nil {
		i.uninstallButton.Enable()
	} else if err == nil {
		i.uninstallButton.Disable()
	} else {
		slog.Warn("Failed to get modInstalled", "error", err)
	}
}

func (i *Installer) Tab() (*container.TabItem, error) {
	bottom := container.NewVBox(
		i.state.ErrorText,
	)
	entry := widget.NewLabelWithData(i.state.SelectedGamePath)
	content := container.New(
		layout.NewBorderLayout(nil, bottom, nil, nil),
		container.NewVScroll(container.NewVBox(
			widget.NewRichTextFromMarkdown("## "+lang.LocalizeKey("installation.select_install_path", "Select Among Us Installation Path")),
			i.state.InstallSelect,
			widget.NewSeparator(),
			widget.NewAccordion(
				widget.NewAccordionItem(lang.LocalizeKey("installation.selected_install", "Selected Installation Path"), container.NewHScroll(container.New(layout.NewCustomPaddedLayout(0, 10, 0, 0), entry))),
			),
			widget.NewSeparator(),
			widget.NewRichTextFromMarkdown("### "+lang.LocalizeKey("installation.installation_status", "Installation Status")), i.state.ModInstalledInfo,
			i.uninstallButton,
			widget.NewSeparator(),
		)),
		bottom,
	)
	return container.NewTabItem(lang.LocalizeKey("installation.tab_name", "Installation"), content), nil
}

func (i *Installer) runUninstall() {
	defer i.state.RefreshModInstallation()
	i.state.ErrorText.Hide()
	path, err := i.state.SelectedGamePath.Get()
	if err != nil || path == "" {
		i.state.ErrorText.Segments = []widget.RichTextSegment{
			&widget.TextSegment{Text: lang.LocalizeKey("installation.error.no_path", "Installation path is not specified."), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
		}
		i.state.ErrorText.Refresh()
		i.state.ErrorText.Show()
		return
	}
	slog.Info("Uninstalling mod", "path", path)

	go func() {
		if err := i.state.Core.UninstallMod(path, i.progressBar); err != nil {
			fyne.Do(func() {
				i.state.ErrorText.Segments = []widget.RichTextSegment{
					&widget.TextSegment{Text: lang.LocalizeKey("installation.error.failed_to_uninstall", "Failed to uninstall mod: ") + err.Error(), Style: widget.RichTextStyle{ColorName: theme.ColorNameError}},
				}
				i.state.ErrorText.Refresh()
				i.state.ErrorText.Show()
				slog.Warn("Failed to uninstall mod", "error", err)
			})
			return
		}
		fyne.Do(func() {
			i.state.ErrorText.ParseMarkdown(lang.LocalizeKey("installation.success.uninstalled", "Mod uninstalled successfully."))
			i.state.ErrorText.Refresh()
			i.state.ErrorText.Show()
			slog.Info("Mod uninstalled successfully", "path", path)
			// i.state.CheckInstalled() was removed in State logic previously?
			// Wait, CheckInstalled is in `client/ui/ui.go` called `state.CheckInstalled()`.
			// `state.go` doesn't seem to have `CheckInstalled` method exposed in my rewrite?
			// Let's check `state.go` again.
			i.state.RefreshModInstallation()
		})
	}()
}