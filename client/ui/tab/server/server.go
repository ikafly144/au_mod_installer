package server

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/sys/windows"

	"github.com/ikafly144/au_mod_installer/client/ui/uicommon"
	"github.com/ikafly144/au_mod_installer/pkg/unityrichtext"
)

const regionInfoFileName = "regioninfo.json"
const regionTypeStatic = "StaticHttpRegionInfo, Assembly-CSharp"
const regionTypeDNS = "DnsRegionInfo, Assembly-CSharp"
const serverIconSize = float32(32)

const globeIconSVG = `<svg xmlns="http://www.w3.org/2000/svg" height="24" viewBox="0 0 24 24" width="24"><path d="M0 0h24v24H0V0z" fill="none"/><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zM4 12c0-.61.08-1.21.21-1.78L8.99 15v1c0 1.1.9 2 2 2v1.93C7.06 19.43 4 16.07 4 12zm13.89 5.4c-.26-.81-1-1.4-1.9-1.4h-1v-3c0-.55-.45-1-1-1h-6v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41C17.92 5.77 20 8.65 20 12c0 2.08-.81 3.98-2.11 5.4z"/></svg>`
const buildCircleIconSVG = `<svg xmlns="http://www.w3.org/2000/svg" enable-background="new 0 0 24 24" height="24" viewBox="0 0 24 24" width="24"><g><rect fill="none" height="24" width="24"/></g><g><g><path d="M12,2C6.48,2,2,6.48,2,12c0,5.52,4.48,10,10,10s10-4.48,10-10 C22,6.48,17.52,2,12,2z M12,20c-4.41,0-8-3.59-8-8c0-4.41,3.59-8,8-8s8,3.59,8,8C20,16.41,16.41,20,12,20z" fill-rule="evenodd"/><path d="M13.49,11.38c0.43-1.22,0.17-2.64-0.81-3.62c-1.11-1.11-2.79-1.3-4.1-0.59 l2.35,2.35l-1.41,1.41L7.17,8.58c-0.71,1.32-0.52,2.99,0.59,4.1c0.98,0.98,2.4,1.24,3.62,0.81l3.41,3.41c0.2,0.2,0.51,0.2,0.71,0 l1.4-1.4c0.2-0.2,0.2-0.51,0-0.71L13.49,11.38z" fill-rule="evenodd"/></g></g></svg>`

var officialServerIcon = theme.NewThemedResource(fyne.NewStaticResource("server-globe.svg", []byte(globeIconSVG)))
var customServerIcon = theme.NewThemedResource(fyne.NewStaticResource("server-build-circle.svg", []byte(buildCircleIconSVG)))

type regionInfo struct {
	CurrentRegionIdx int               `json:"CurrentRegionIdx"`
	Regions          []json.RawMessage `json:"Regions"`
}

type staticRegionInfo struct {
	Type          string   `json:"$type,omitempty"`
	Name          string   `json:"Name"`
	PingServer    string   `json:"PingServer"`
	Servers       []ipPort `json:"Servers"`
	TranslateName int      `json:"TranslateName"`
	UseDtls       bool     `json:"UseDtls"`
	Port          uint16   `json:"Port"`
}

type dnsRegionInfo struct {
	Type          string  `json:"$type,omitempty"`
	Fqdn          string  `json:"Fqdn"`
	DefaultIP     *string `json:"DefaultIp"`
	Port          uint16  `json:"Port"`
	UseDtls       bool    `json:"UseDtls"`
	Name          string  `json:"Name"`
	TranslateName int     `json:"TranslateName"`
	TargetServer  *string `json:"TargetServer"`
}

type ipPort struct {
	Ip   string `json:"Ip"`
	Port uint16 `json:"Port"`
}

type regionEntry struct {
	Type          string
	Name          string
	TranslateName int
	UseDtls       bool
	Port          uint16

	PingServer string
	Servers    []ipPort

	Fqdn         string
	DefaultIP    *string
	TargetServer *string
}

type ServerTab struct {
	state *uicommon.State

	list      *widget.List
	addButton *widget.Button
	bodyStack *fyne.Container
	emptyHint *widget.Label

	regions          []regionEntry
	currentRegionIdx int
	regionInfoPath   string
}

func NewServerTab(state *uicommon.State) *ServerTab {
	s := &ServerTab{
		state:            state,
		currentRegionIdx: 0,
	}
	s.addButton = widget.NewButtonWithIcon(lang.LocalizeKey("server.add", "Add"), theme.ContentAddIcon(), s.showAddDialog)
	s.addButton.Importance = widget.HighImportance
	s.list = widget.NewList(
		func() int { return len(s.regions) },
		func() fyne.CanvasObject {
			icon := widget.NewIcon(customServerIcon)
			iconHolder := container.New(
				layout.NewGridWrapLayout(fyne.NewSquareSize(serverIconSize)),
				icon,
			)
			name := widget.NewRichText(&widget.TextSegment{
				Style: widget.RichTextStyleStrong,
				Text:  "",
			})
			name.Wrapping = fyne.TextWrapWord
			address := widget.NewLabel("")
			address.Wrapping = fyne.TextWrapWord
			editBtn := widget.NewButtonWithIcon(lang.LocalizeKey("server.update", "Update"), theme.DocumentCreateIcon(), nil)
			deleteBtn := widget.NewButtonWithIcon(lang.LocalizeKey("server.delete", "Delete"), theme.DeleteIcon(), nil)
			deleteBtn.Importance = widget.DangerImportance
			editBtn.Importance = widget.LowImportance
			deleteBtn.Importance = widget.LowImportance
			body := container.NewVBox(name, address)
			actions := container.NewHBox(editBtn, deleteBtn)
			content := container.New(&serverListItemLayout{
				minIconSize: serverIconSize,
				spacing:     theme.Padding(),
			}, iconHolder, body, actions)

			bg := canvas.NewRectangle(color.Transparent)
			bg.CornerRadius = theme.InputRadiusSize()
			bg.StrokeColor = theme.Color(theme.ColorNameButton)
			bg.StrokeWidth = 1
			return container.NewStack(
				bg,
				container.NewPadded(content),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			if id < 0 || id >= len(s.regions) {
				return
			}
			rowStack := obj.(*fyne.Container)
			row := rowStack.Objects[1].(*fyne.Container).Objects[0].(*fyne.Container)
			current := s.regions[id]
			iconHolder := row.Objects[0].(*fyne.Container)
			icon := iconHolder.Objects[0].(*widget.Icon)
			if current.isOfficial() {
				icon.SetResource(officialServerIcon)
			} else {
				icon.SetResource(customServerIcon)
			}

			body := row.Objects[1].(*fyne.Container)
			name := body.Objects[0].(*widget.RichText)
			name.Segments = unityrichtext.Parse(current.Name)
			name.Refresh()

			body.Objects[1].(*widget.Label).SetText(current.address())

			actions := row.Objects[2].(*fyne.Container)
			editBtn := actions.Objects[0].(*widget.Button)
			deleteBtn := actions.Objects[1].(*widget.Button)
			editBtn.OnTapped = func() {
				if current.isOfficial() {
					return
				}
				s.showEditDialog(id)
			}
			if current.isOfficial() {
				editBtn.Disable()
				deleteBtn.Disable()
			} else {
				editBtn.Enable()
				deleteBtn.Enable()
			}
			deleteBtn.OnTapped = func() {
				s.deleteServer(id)
			}
		},
	)
	s.list.HideSeparators = true
	if err := s.loadFromDisk(); err != nil {
		s.state.ShowErrorDialog(err)
	}
	return s
}

func (s *ServerTab) Tab() (*container.TabItem, error) {
	footer := container.NewPadded(container.NewBorder(
		nil, nil, nil, nil, s.addButton,
	))
	s.emptyHint = widget.NewLabel(lang.LocalizeKey("server.ui.empty", "No servers. Add one."))
	s.emptyHint.Wrapping = fyne.TextWrapWord
	s.bodyStack = container.NewStack(
		container.NewCenter(s.emptyHint),
		s.list,
	)
	s.refreshView()
	content := container.NewBorder(nil, footer, nil, nil, container.NewPadded(s.bodyStack))
	return container.NewTabItem(lang.LocalizeKey("server.tab_name", "Server"), content), nil
}

func (s *ServerTab) showAddDialog() {
	s.showServerDialog(
		lang.LocalizeKey("server.add", "Add"),
		regionEntry{
			Type:          regionTypeStatic,
			TranslateName: 1003,
		},
		true,
		func(entry regionEntry) error {
			s.regions = append(s.regions, entry)
			return s.saveToDisk()
		},
	)
}

func (s *ServerTab) showEditDialog(index int) {
	if index < 0 || index >= len(s.regions) {
		return
	}
	current := s.regions[index]
	if current.isOfficial() {
		return
	}
	s.showServerDialog(
		lang.LocalizeKey("server.update", "Update"),
		current,
		false,
		func(entry regionEntry) error {
			s.regions[index] = entry
			return s.saveToDisk()
		},
	)
}

func (s *ServerTab) showServerDialog(title string, current regionEntry, isNew bool, onSave func(regionEntry) error) {
	ip, port := current.primaryHostPort()
	nameEntry := widget.NewEntry()
	nameEntry.MultiLine = false
	nameEntry.Wrapping = fyne.TextWrapOff
	nameEntry.SetText(current.Name)
	nameEntry.SetPlaceHolder(lang.LocalizeKey("server.form.name_placeholder", "Region Name (e.g. Japan)"))
	ipEntry := widget.NewEntry()
	ipEntry.MultiLine = false
	ipEntry.Wrapping = fyne.TextWrapOff
	ipEntry.SetText(ip)
	ipEntry.SetPlaceHolder(lang.LocalizeKey("server.form.ip_placeholder", "IP Address or Hostname"))
	portEntry := widget.NewEntry()
	portEntry.MultiLine = false
	portEntry.Wrapping = fyne.TextWrapOff
	if port > 0 {
		portEntry.SetText(fmt.Sprintf("%d", port))
	}
	portEntry.SetPlaceHolder(lang.LocalizeKey("server.form.port_placeholder", "22023"))
	items := []*widget.FormItem{
		widget.NewFormItem(lang.LocalizeKey("server.form.name", "Name"), nameEntry),
		widget.NewFormItem(lang.LocalizeKey("server.form.ip", "IP / Host"), ipEntry),
		widget.NewFormItem(lang.LocalizeKey("server.form.port", "Port"), portEntry),
	}
	d := dialog.NewForm(
		title,
		lang.LocalizeKey("common.save", "Save"),
		lang.LocalizeKey("common.cancel", "Cancel"),
		items,
		func(ok bool) {
			if !ok {
				return
			}
			entry, err := buildServerEntry(nameEntry.Text, ipEntry.Text, portEntry.Text, current, isNew)
			if err != nil {
				s.state.ShowErrorDialog(err)
				return
			}
			if err := onSave(entry); err != nil {
				s.state.ShowErrorDialog(err)
				return
			}
			s.list.Refresh()
			s.refreshView()
		},
		s.state.Window,
	)
	d.Resize(fyne.NewSize(640, 360))
	d.Show()
}

func (s *ServerTab) deleteServer(index int) {
	if index < 0 || index >= len(s.regions) {
		return
	}
	dialog.ShowConfirm(
		lang.LocalizeKey("server.delete_confirm_title", "Delete server"),
		lang.LocalizeKey("server.delete_confirm_message", "Delete selected server entry?"),
		func(ok bool) {
			if !ok {
				return
			}
			s.regions = append(s.regions[:index], s.regions[index+1:]...)
			if s.currentRegionIdx > index {
				s.currentRegionIdx--
			} else if s.currentRegionIdx == index {
				s.currentRegionIdx = 0
			}
			if err := s.saveToDisk(); err != nil {
				s.state.ShowErrorDialog(err)
				return
			}
			s.list.Refresh()
			s.refreshView()
		},
		s.state.Window,
	)
}

func (s *ServerTab) loadFromDisk() error {
	path, err := resolveRegionInfoPath()
	if err != nil {
		return err
	}
	s.regionInfoPath = path
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			s.regions = nil
			s.currentRegionIdx = 0
			return s.saveToDisk()
		}
		return fmt.Errorf("%s: %w", lang.LocalizeKey("server.error.read_failed", "Failed to read regioninfo.json"), err)
	}
	var info regionInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return fmt.Errorf("%s: %w", lang.LocalizeKey("server.error.parse_failed", "Failed to parse regioninfo.json"), err)
	}
	s.regions = s.regions[:0]
	for _, regionRaw := range info.Regions {
		entry, decodeErr := decodeRegionEntry(regionRaw)
		if decodeErr != nil {
			return fmt.Errorf("%s: %w", lang.LocalizeKey("server.error.parse_failed", "Failed to parse regioninfo.json"), decodeErr)
		}
		s.regions = append(s.regions, entry)
	}
	s.currentRegionIdx = info.CurrentRegionIdx
	s.list.Refresh()
	s.refreshView()
	return nil
}

func (s *ServerTab) saveToDisk() error {
	if s.regionInfoPath == "" {
		path, err := resolveRegionInfoPath()
		if err != nil {
			return err
		}
		s.regionInfoPath = path
	}
	currentIdx := s.currentRegionIdx
	if len(s.regions) == 0 || currentIdx < 0 || currentIdx >= len(s.regions) {
		currentIdx = 0
	}
	regions := make([]any, 0, len(s.regions))
	for _, entry := range s.regions {
		regions = append(regions, entry.toJSON())
	}
	payload := regionInfo{
		CurrentRegionIdx: currentIdx,
		Regions:          make([]json.RawMessage, 0, len(regions)),
	}
	for _, r := range regions {
		raw, err := json.Marshal(r)
		if err != nil {
			return err
		}
		payload.Regions = append(payload.Regions, raw)
	}
	raw, err := json.MarshalIndent(payload, "", "    ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	if err := os.MkdirAll(filepath.Dir(s.regionInfoPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(s.regionInfoPath, raw, 0644)
}

func resolveRegionInfoPath() (string, error) {
	localAppDataLow, err := windows.KnownFolderPath(windows.FOLDERID_LocalAppDataLow, 0)
	if err != nil {
		return "", err
	}
	path := filepath.Join(localAppDataLow, "Innersloth", "Among Us", regionInfoFileName)
	if strings.TrimSpace(path) == "" {
		return "", fmt.Errorf("%s", lang.LocalizeKey("server.error.region_path_not_found", "Could not resolve regioninfo.json path."))
	}
	return path, nil
}

func buildServerEntry(nameText, ipText, portText string, base regionEntry, isNew bool) (regionEntry, error) {
	name := strings.TrimSpace(nameText)
	ip := strings.TrimSpace(ipText)
	portRaw := strings.TrimSpace(portText)
	if name == "" {
		return regionEntry{}, fmt.Errorf("%s", lang.LocalizeKey("server.error.name_required", "Name is required."))
	}
	if ip == "" {
		return regionEntry{}, fmt.Errorf("%s", lang.LocalizeKey("server.error.ip_required", "IP/Host is required."))
	}
	var port uint16 = 22023
	if portRaw != "" {
		parsed, err := strconv.ParseUint(portRaw, 10, 16)
		if err != nil || parsed == 0 {
			return regionEntry{}, fmt.Errorf("%s", lang.LocalizeKey("server.error.invalid_port", "Port must be a number between 1 and 65535."))
		}
		port = uint16(parsed)
	}
	if isNew {
		base.Type = regionTypeStatic
		base.TranslateName = 1003
		base.UseDtls = true
	}
	if base.Type == "" {
		base.Type = regionTypeStatic
	}

	base.Name = name
	base.Port = port
	if base.TranslateName == 0 {
		base.TranslateName = 1003
	}
	if base.isDNSLike() {
		base.Fqdn = ip
		base.PingServer = ""
		base.Servers = nil
	} else {
		base.PingServer = ip
		if len(base.Servers) == 0 {
			base.Servers = []ipPort{{}}
		}
		base.Servers[0] = ipPort{Ip: ip, Port: port}
		base.Fqdn = ""
		base.DefaultIP = nil
		base.TargetServer = nil
	}
	return base, nil
}

func decodeRegionEntry(raw json.RawMessage) (regionEntry, error) {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return regionEntry{}, err
	}
	var typeName string
	if typeRaw, ok := probe["$type"]; ok {
		_ = json.Unmarshal(typeRaw, &typeName)
	}
	_, hasFQDN := probe["Fqdn"]
	if hasFQDN || strings.Contains(strings.ToLower(typeName), "dnsregioninfo") {
		var dns dnsRegionInfo
		if err := json.Unmarshal(raw, &dns); err != nil {
			return regionEntry{}, err
		}
		if dns.Type == "" {
			dns.Type = regionTypeDNS
		}
		return regionEntry{
			Type:          dns.Type,
			Name:          dns.Name,
			TranslateName: dns.TranslateName,
			UseDtls:       dns.UseDtls,
			Port:          dns.Port,
			Fqdn:          strings.TrimSpace(dns.Fqdn),
			DefaultIP:     dns.DefaultIP,
			TargetServer:  dns.TargetServer,
		}, nil
	}
	var st staticRegionInfo
	if err := json.Unmarshal(raw, &st); err != nil {
		return regionEntry{}, err
	}
	if st.Type == "" {
		st.Type = regionTypeStatic
	}
	entry := regionEntry{
		Type:          st.Type,
		Name:          st.Name,
		TranslateName: st.TranslateName,
		UseDtls:       st.UseDtls,
		Port:          st.Port,
		PingServer:    strings.TrimSpace(st.PingServer),
		Servers:       append([]ipPort(nil), st.Servers...),
	}
	if len(entry.Servers) == 0 && entry.PingServer != "" {
		entry.Servers = []ipPort{{Ip: entry.PingServer, Port: entry.Port}}
	}
	return entry, nil
}

func (r regionEntry) toJSON() any {
	if r.isDNSLike() {
		return dnsRegionInfo{
			Type:          r.withDefaultType(regionTypeDNS),
			Fqdn:          strings.TrimSpace(r.Fqdn),
			DefaultIP:     r.DefaultIP,
			Port:          r.Port,
			UseDtls:       r.UseDtls,
			Name:          r.Name,
			TranslateName: r.TranslateName,
			TargetServer:  r.TargetServer,
		}
	}
	servers := append([]ipPort(nil), r.Servers...)
	if len(servers) == 0 && r.PingServer != "" {
		servers = []ipPort{{Ip: r.PingServer, Port: r.Port}}
	}
	return staticRegionInfo{
		Type:          r.withDefaultType(regionTypeStatic),
		Name:          r.Name,
		PingServer:    strings.TrimSpace(r.PingServer),
		Servers:       servers,
		TranslateName: r.TranslateName,
		UseDtls:       r.UseDtls,
		Port:          r.Port,
	}
}

func (r regionEntry) withDefaultType(fallback string) string {
	if strings.TrimSpace(r.Type) == "" {
		return fallback
	}
	return r.Type
}

func (r regionEntry) isDNSLike() bool {
	return strings.Contains(strings.ToLower(r.Type), "dnsregioninfo") || r.Fqdn != ""
}

func (r regionEntry) primaryHostPort() (string, uint16) {
	if r.isDNSLike() {
		return strings.TrimSpace(r.Fqdn), r.Port
	}
	if len(r.Servers) > 0 {
		return strings.TrimSpace(r.Servers[0].Ip), r.Servers[0].Port
	}
	return strings.TrimSpace(r.PingServer), r.Port
}

func (r regionEntry) address() string {
	ip, port := r.primaryHostPort()
	if ip == "" {
		return "-"
	}
	if port == 0 {
		return ip
	}
	return fmt.Sprintf("%s:%d", ip, port)
}

func (r regionEntry) isOfficial() bool {
	name := strings.ToLower(strings.TrimSpace(r.plainName()))
	if name == "custom" {
		return false
	}
	return r.TranslateName != 1003
}

func (r regionEntry) plainName() string {
	var b strings.Builder
	for _, seg := range unityrichtext.Parse(r.Name) {
		b.WriteString(seg.Textual())
	}
	return b.String()
}

func (s *ServerTab) refreshView() {
	if s.emptyHint == nil || s.bodyStack == nil {
		return
	}
	if len(s.regions) == 0 {
		s.emptyHint.Show()
		s.list.Hide()
	} else {
		s.emptyHint.Hide()
		s.list.Show()
	}
	s.bodyStack.Refresh()
}

type serverListItemLayout struct {
	minIconSize float32
	spacing     float32
}

func (l *serverListItemLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) < 3 {
		return
	}
	icon := objects[0]
	body := objects[1]
	actions := objects[2]

	actionsMin := actions.MinSize()
	actionsX := size.Width - actionsMin.Width
	if actionsX < 0 {
		actionsX = 0
	}
	actionsY := (size.Height - actionsMin.Height) / 2
	if actionsY < 0 {
		actionsY = 0
	}
	actions.Resize(actionsMin)
	actions.Move(fyne.NewPos(actionsX, actionsY))

	iconSide := l.minIconSize
	if iconSide > size.Height {
		iconSide = size.Height
	}
	icon.Resize(fyne.NewSquareSize(iconSide))
	iconY := (size.Height - iconSide) / 2
	if iconY < 0 {
		iconY = 0
	}
	icon.Move(fyne.NewPos(0, iconY))

	bodyX := iconSide + l.spacing
	bodyWidth := actionsX - l.spacing - bodyX
	if bodyWidth < 0 {
		bodyWidth = 0
	}
	body.Resize(fyne.NewSize(bodyWidth, size.Height))
	body.Move(fyne.NewPos(bodyX, 0))
}

func (l *serverListItemLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) < 3 {
		return fyne.NewSize(0, 0)
	}
	iconMin := objects[0].MinSize()
	bodyMin := objects[1].MinSize()
	actionsMin := objects[2].MinSize()

	iconSide := iconMin.Height
	if iconSide < l.minIconSize {
		iconSide = l.minIconSize
	}
	height := iconSide
	if bodyMin.Height > height {
		height = bodyMin.Height
	}
	if actionsMin.Height > height {
		height = actionsMin.Height
	}
	width := iconSide + l.spacing + bodyMin.Width + l.spacing + actionsMin.Width
	return fyne.NewSize(width, height)
}
