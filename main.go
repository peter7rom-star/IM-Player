package main

import (
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/diamondburned/gotk4/pkg/gdk/v3"
	"github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/gen2brain/dlgs"
	"golang.org/x/exp/slices"
	"gopkg.in/vansante/go-ffprobe.v2"

	stream_db "github.com/tech7strann1k/online-radio/db"
)

var homeDir, _ = os.UserHomeDir()
var click, ind int
var builder = gtk.NewBuilder()
var player = NewPlayer()
var wnd *MainWindow
var resource_path = fmt.Sprintf("%s/%s", homeDir, ".local/share/online-radio")
var db = stream_db.InitDB(fmt.Sprintf("%s/db/metadata.db", resource_path))
var state, playlistState, playlistViewState string
var forward = false
var currentPlaylistItemIndex interface{}
var selectedStreamList []stream_db.StreamItem
var favList []stream_db.FavouriteItem
var streamItem stream_db.StreamItem
var favouriteItem stream_db.FavouriteItem
var currentTitle []string


type MainWindow struct {
	MainWindow   			*gtk.Window
	ViewPort     			*gtk.Viewport
	PlaylistView 			*gtk.ListBox
	PlaylistViewTable		*gtk.Table
	PlayButton, StopButton,
	AddButton, PrefsButton,
	LibButton, FavButton,
	RecordButton 			*gtk.Button
	PlayImg, StopImg,
	LibImg, FavImg,
	RecordImg *gtk.Image
	SelectCountryBox 		*gtk.ComboBoxText
	MetadataView     		*gtk.Label
	StreamLogoView   		*gtk.Image
	Player           		*StreamPlayer
}

type AddStreamDialog struct {
	Dialog                                              *gtk.Dialog
	AddStreamNameBox, AddStreamUrlBox, AddStreamIconBox *gtk.Entry
	AddStreamIconButton, OkButton, CancelButton         *gtk.Button
}

type StreamPropertiesDialog struct {
	Dialog								*gtk.Dialog
	StreamNameBox, 
	StreamUrlBox						*gtk.Entry
	StreamBitrateBox,	
	StreamCountryBox					*gtk.Label
	OkButton, CancelButton				*gtk.Button
}

type SettingsDialog struct {
	Dialog								*gtk.Dialog
	InterfaceBox, DefaultViewBox		*gtk.ComboBoxText
	AboutButton, OkButton, CancelButton	*gtk.Button
}

type SettingsData struct {
	DefaultViewState				string `json: "default_view`
}

func NewMainWindow() *MainWindow {
	builder = gtk.NewBuilderFromFile(fmt.Sprintf("%s/online_radio_app.glade", resource_path))
	window := builder.GetObject("window_main").Cast().(*gtk.Window)
	window.SetTitle("Online Radio")
	iconName := fmt.Sprintf("%s/radio_icon.png", resource_path)
	pixbuf, _ := gdkpixbuf.NewPixbufFromFile(iconName)
	window.SetIcon(pixbuf)
	window.SetDefaultSize(720, 720)
	viewPort := builder.GetObject("view_port").Cast().(*gtk.Viewport)
	playButton := builder.GetObject("play_button").Cast().(*gtk.Button)
	playImg := builder.GetObject("play_img").Cast().(*gtk.Image)
	stopButton := builder.GetObject("stop_button").Cast().(*gtk.Button)
	stopImg := builder.GetObject("stop_img").Cast().(*gtk.Image)
	recordButton := builder.GetObject("record_button").Cast().(*gtk.Button)
	recordImg := builder.GetObject("record_img").Cast().(*gtk.Image)
	metadataView := builder.GetObject("stream_metadata_label").Cast().(*gtk.Label)
	streamLogoView := builder.GetObject("stream_logo_view").Cast().(*gtk.Image)
	addButton := builder.GetObject("add_button").Cast().(*gtk.Button)
	prefsButton := builder.GetObject("prefs_button").Cast().(*gtk.Button)
	selectCountryBox := builder.GetObject("select_country_box").Cast().(*gtk.ComboBoxText)
	libButton := builder.GetObject("lib_button").Cast().(*gtk.Button)
	libImg := builder.GetObject("lib_img").Cast().(*gtk.Image)
	favButton := builder.GetObject("fav_button").Cast().(*gtk.Button)
	favImg := builder.GetObject("fav_img").Cast().(*gtk.Image)
	mainWindow := &MainWindow{MainWindow: window, ViewPort: viewPort, PlayButton: playButton,
		PlayImg: playImg, StopButton: stopButton, StopImg: stopImg, MetadataView: metadataView,
		StreamLogoView: streamLogoView, AddButton: addButton, PrefsButton: prefsButton,
		SelectCountryBox: selectCountryBox, LibButton: libButton, LibImg: libImg,
		FavButton: favButton, FavImg: favImg, RecordButton: recordButton, RecordImg: recordImg}
	return mainWindow
}

func (wnd *MainWindow) Activate(app *gtk.Application) {
	wnd.SelectCountryBox.AppendText("All")
	var query, query_2 string
	wnd.SelectCountryBox.SetActive(0)
	query = wnd.SelectCountryBox.ActiveText()
	player.StreamList = db.LoadStationList(nil)
	favouritesList := db.LoadFavourites()
	favList = db.LoadFavourites()
	state = "default"
	input := os.ReadFile("settings.json")
	var settingsData SettingsData
	jsonData = json.Unmarshal(input, &settingsData)
	if settingsData.DefaultViewState == "List" {
		data := player.StreamList
	} else {
		data := favouritesList
	}
	rowsQuantity := uint(len(data))
	wnd.PlaylistView = gtk.NewListBox()
	wnd.PlaylistViewTable = gtk.NewTable(rowsQuantity, uint(4), true)
	listView_2 := gtk.NewListBox()
	listView_3 := gtk.NewListBox()
	var m sync.Mutex
	for ind, item := range data {
		wnd.AddItemToPlaylist(ind, item)
	}
	listView := wnd.PlaylistView
	// box := gtk.NewBox(gtk.OrientationHorizontal, 2)
	// box.Add(wnd.PlaylistView)
	wnd.ViewPort.Add(wnd.PlaylistView)
	wnd.PlaylistView.ConnectSelectionClearEvent(func(event *gdk.EventSelection) (ok bool) {
		selectedStreamList = nil
		return
	})
	player.playing_state = player.Stopped
	d := time.Duration.Milliseconds(10)
	wnd.PlayButton.ConnectClicked(func() {
		click++
		if len(selectedStreamList) > 0 {
			go wnd.SelectedRowHandler(selectedStreamList[0])
			
		} else if len(selectedFavList) > 0 {
			go wnd.SelectedRowHandler(selectedFavList[0])
		}
		if click == 1 {
			player.playing_state = player.Started
			go func() {
				player.Play()
				wnd.updateMetadata()
			}()
		} else {
			m.Lock()
			go func() {
				player.StopPlayback()
				player.playing_state = player.Playing
				player.Play()
			}()
			m.Unlock()
		}
	})
	wnd.StopButton.ConnectClicked(func() {
		go func() {
			if player.playing_state != player.Stopped {
				time.Sleep(time.Duration(d))
				player.playing_state = player.Stopped
				player.StopPlayback()
			}
		}()
	})
	wnd.AddButton.ConnectClicked(func() {
		dlg := NewAddStreamDialog()
		dlg.Init()
		dlg.Dialog.ShowAll()
	})
	var landList = db.LoadLandList()
	model := append(landList, "favourites")
	for _, elem := range model {
		wnd.SelectCountryBox.AppendText(elem)
	}
	wnd.SelectCountryBox.SetCanFocus(true)
	wnd.FavButton.ConnectClicked(func() {
		
		wnd.PlaylistView = listView_2
		state = "favourites_selected"
		ind := slices.Index(model, "favourites")
		ind++
		wnd.SelectCountryBox.SetActive(ind)
	})
	wnd.LibButton.ConnectClicked(func() {
		state = "library_selected"
		ind := slices.Index(model, query_2)
		ind++
		wnd.SelectCountryBox.SetActive(ind)
	})
	wnd.RecordButton.ConnectClicked(func() {
		go player.RecordStream()
	})
	wnd.PlayImg.SetFromFile(fmt.Sprintf("%s/%s", resource_path, "play.png"))
	wnd.StopImg.SetFromFile(fmt.Sprintf("%s/%s", resource_path, "stop.png"))
	wnd.RecordImg.SetFromFile(fmt.Sprintf("%s/%s", resource_path, "record.png"))
	wnd.LibImg.SetFromFile(fmt.Sprintf("%s/%s", resource_path, "library.png"))
	wnd.FavImg.SetFromFile(fmt.Sprintf("%s/%s", resource_path, "favorite_icon.png"))
	wnd.SelectCountryBox.ConnectChanged(func() {
		// if currentPlaylistItemIndex != nil {
		// 	row := wnd.PlaylistView.RowAtIndex(currentPlaylistItemIndex.(int))
		// 	wnd.PlaylistView.SelectRow(row)
		// }
		wnd.ViewPort.Remove(wnd.ViewPort.Children()[0])
		query = wnd.SelectCountryBox.ActiveText()
		fmt.Println(favouriteItem.StreamName.IsZero())
		state = "default"
		selectedStreamList = []stream_db.StreamItem{}
		selectedFavList = []stream_db.FavouriteItem{}
		if query == "All" || len(query) == 0 {
			wnd.PlaylistView = listView
			query_2 = query
		} else {
			wnd.PlaylistView = gtk.NewListBox()
			if query == "favourites"  {
				state = "favourites_selected"
				favList = db.LoadFavourites()
				if len(listView_3.Children()) > 0 {
					wnd.PlaylistView = listView_3
					fv, _ := db.GetFavouritesByItemName(favouriteItem.StreamName.String)
					row := wnd.PlaylistView.RowAtIndex(int(fv.Id.Int64))
					wnd.PlaylistView.SelectRow(row)
					for ind, item := range favList {
						row := wnd.PlaylistView.RowAtIndex(ind)
						convItem := item.ToStream()
						row.ConnectButtonPressEvent(func(event *gdk.EventButton) (ok bool) {
							return wnd.onRowClickHandler(ind, convItem, event)
						})
					}	
				} else {
					for _, item := range favList {
						convItem := item.ToStream()
						wnd.AddItemToPlaylist(nil, convItem)
					}
					listView_3 = wnd.PlaylistView
				}
			} else {
				query_2 = query
				if slices.Contains(landList, query) {
					player.StreamList = db.LoadStationListFromCountry(query)
				} else {
					player.StreamList = db.LoadStationList(query)
				}
				if state == "library_selected" {
					wnd.PlaylistView = listView_2
					
					state = "default"
				}
			}
			if state == "default" {
				for ind, item := range player.StreamList {
					wnd.AddItemToPlaylist(ind, item)
				}
				listView_2 = wnd.PlaylistView
			}
			currentPlaylistItemIndex = nil
			// state = "default"
		}
		wnd.PlaylistView.ShowAll()
		wnd.ViewPort.Add(wnd.PlaylistView)
	})
	// playlistView.ConnectButtonPressEvent(func(event *gdk.EventButton) (ok bool) {

	// })
	window := wnd.MainWindow
	app.AddWindow(window)
	window.SetIconFromFile(fmt.Sprintf(
		"%s/radio_icon.png", resource_path))
	window.ShowAll()
}

func NewAddStreamDialog() *AddStreamDialog {
	builder = gtk.NewBuilderFromFile(fmt.Sprintf("%s/online_radio_app.glade", resource_path))
	dialog := builder.GetObject("add_stream_dialog").Cast().(*gtk.Dialog)
	dialog.SetTitle("Add Stream")
	addStreamNameBox := builder.GetObject("add_stream_name_box").Cast().(*gtk.Entry)
	addStreamUrlBox := builder.GetObject("add_stream_url_box").Cast().(*gtk.Entry)
	addStreamIconBox := builder.GetObject("add_stream_icon_box").Cast().(*gtk.Entry)
	addStreamIconButton := builder.GetObject("add_stream_icon_button").Cast().(*gtk.Button)
	okButton := builder.GetObject("ok_button").Cast().(*gtk.Button)
	cancelButton := builder.GetObject("cancel_button").Cast().(*gtk.Button)
	addStreamDialog := &AddStreamDialog{Dialog: dialog, AddStreamNameBox: addStreamNameBox, AddStreamUrlBox: addStreamUrlBox,
		AddStreamIconBox: addStreamIconBox, AddStreamIconButton: addStreamIconButton,
		OkButton: okButton, CancelButton: cancelButton}
	return addStreamDialog
}

func (dlg *AddStreamDialog) Init() {
	var initPath string
	dlg.AddStreamIconButton.ConnectClicked(func() {
		file, _, err := dlgs.File("Open file", "*.png *.jpg", false)
		if err != nil {
			fmt.Println(err)
		}
		initPath = file
		dlg.AddStreamIconBox.SetText(initPath)
	})
	dlg.OkButton.ConnectReleased(func() {
		var dirs = strings.Split(initPath, "/")
		var filename = dirs[len(dirs)-1]
		os.Chdir(resource_path)
		destPath := fmt.Sprintf("./%s", filename)
		_, err := os.Stat(destPath)
		if os.IsNotExist(err) {
			initFile, _ := os.Open(initPath)
			defer initFile.Close()
			destFile, _ := os.Create(destPath)
			defer destFile.Close()
			io.Copy(initFile, destFile)
		}
		streamName := dlg.AddStreamNameBox.Text()
		streamUrl := dlg.AddStreamUrlBox.Text()
		db.AddToFavourites(streamName, streamUrl, filename)
		item, _ := db.GetFavouritesByItemName(streamName)
		convItem := item.ToStream()
		wnd.AddItemToPlaylist(nil, convItem)
		dlg.Dialog.Close()
	})
	dlg.CancelButton.ConnectReleased(func() {
		dlg.Dialog.Close()
	})

}

func NewStreamPropertiesDialog() *StreamPropertiesDialog {
	dialog := builder.GetObject("stream_properties_dialog").Cast().(*gtk.Dialog)
	streamNameBox := builder.GetObject("stream_name_box").Cast().(*gtk.Entry)
	streamUrlBox := builder.GetObject("stream_url_box").Cast().(*gtk.Entry)
	streamBitrateBox := builder.GetObject("stream_bitrate_box").Cast().(*gtk.Label)
	streamCountryBox := builder.GetObject("stream_country_box").Cast().(*gtk.Label)
	okButton := builder.GetObject("ok").Cast().(*gtk.Button)
	cancelButton := builder.GetObject("cancel").Cast().(*gtk.Button)
	return &StreamPropertiesDialog{Dialog: dialog, StreamNameBox: streamNameBox, 
		StreamUrlBox: streamUrlBox, StreamBitrateBox: streamBitrateBox, StreamCountryBox: streamCountryBox, 
		OkButton: okButton, CancelButton: cancelButton,}
}

func (dlg *StreamPropertiesDialog) Init(item stream_db.StreamItem) {
	oldStreamName := item.StreamName.String
	dlg.StreamNameBox.SetText(item.StreamName.String)
	dlg.StreamUrlBox.SetText(item.Url.String)
	metadata := getMetadata()
	bitrate := metadata.BitRate[:3]
	bitrate = fmt.Sprintf("%s %s", bitrate, "kb/s")
	dlg.StreamBitrateBox.SetText(bitrate)
	dlg.StreamCountryBox.SetText(item.Country.String)	
	dlg.OkButton.ConnectReleased(func() {
		db.Update(oldStreamName, dlg.StreamNameBox.
			Text(), dlg.StreamUrlBox.Text())
			dlg.Dialog.Close()
	})
	dlg.CancelButton.ConnectReleased(func ()  {
		dlg.Dialog.Close()
	})
	
	
}

func NewSettingsDialog() *SettingsDialog {
	dialog := builder.GetObject("settings_dialog").Cast().(*gtk.Dialog)
	interfaceBox := builder.GetObject("interface_box").Cast().(*gtk.ComboBoxText)
	defaultViewBox := builder.GetObject("default_view_box").Cast().(*gtk.ComboBoxText)
	okButton := builder.GetObject("ok_butto").Cast().(*gtk.Button)
	cancelButton := builder.GetObject("cancel_butto").Cast().(*gtk.Button)
	return &SettingsDialog{Dialog: dialog, InterfaceBox: interfaceBox, DefaultViewBox: defaultViewBox, 
						   OkButton: okButton, CancelButton: cancelButton,}
}

func (dlg *SettingsDialog) Init() {
	dlg.DefaultViewBox.ConnectChanged(func() {
		defaultViewState := dlg.DefaultViewBox.ActiveText()
		data := SettingsData{DefaultViewState: defaultviewst}
		output = json.Marshal(data)
		os.WriteFile("settings.json", output, os.ModePerm)
	})

}

func main() {
	gtk.Init()
	wnd = NewMainWindow()
	app := gtk.NewApplication("com.github.tech7strann1k.online-radio", 0)
	app.ConnectActivate(func() { wnd.Activate(app) })
	app.ConnectShutdown(func() {
		if player.playing_state == player.Playing {
			time.Sleep(3 * time.Millisecond)
			player.StopPlayback()
		}
	})
	if code := app.Run(os.Args); code > 0 {
		if player.playing_state == player.Playing {
			time.Sleep(3 * time.Millisecond)
			player.StopPlayback()
		}
		os.Exit(code)
	}
}

func getMetadata() ffprobe.Format {
	var m sync.Mutex
	ch := make(chan *ffprobe.Format)
	m.Lock()
	go player.GetMetadata(ch)
	metadata := ffprobe.Format(*<-ch)
	m.Unlock()
	return metadata
}

var selectedFavList []stream_db.FavouriteItem

func addRow(item stream_db.StreamItem) (*gtk.EventBox, *gtk.ListBoxRow, stream_db.StreamItem) {
	hbox := gtk.NewBox(gtk.OrientationHorizontal, 7)
	eventBox := gtk.NewEventBox()
	var streamLogo = fmt.Sprintf("%s/./radio_logos/%s", resource_path, item.Logo.String)
	var streamLogo_2 = fmt.Sprintf("%s/radio.png", resource_path)
	_, err := os.Stat(streamLogo)
	if os.IsNotExist(err) || item.Logo.String == "" {
		streamLogo = streamLogo_2
	}
	// width, height, _ := gdkpixbuf.PixbufGetFileInfo(streamLogo)
	// if width != 32 && height != 32 {
	// 	streamLogo = streamLogo_2
	// }
	logoImage := gtk.NewImageFromFile(streamLogo)
	logoLabel := gtk.NewLabel(item.StreamName.String)
	hbox.Add(logoImage)
	hbox.Add(logoLabel)
	eventBox.Add(hbox)
	row := gtk.NewListBoxRow()
	row.Add(eventBox)
	return eventBox, row, item
}

func (wnd *MainWindow) AddItemToPlaylist(index any, item stream_db.StreamItem) {
	_, row, item := addRow(item)
	wnd.PlaylistView.Add(row)
	wnd.PlaylistViewTable.Add(row)
	row.ConnectButtonPressEvent(func(event *gdk.EventButton) (ok bool) {
		wnd.onRowClickHandler(index, item, event)
		return
	}))
}

func (wnd *MainWindow) onRowClickHandler(index any, item stream_db.StreamItem, event *gdk.EventButton) (ok bool) {
	selectedStreamList = []stream_db.StreamItem{}
	selectedFavList = []stream_db.FavouriteItem{}
	if state == "default" {
		stream, _ := db.GetStreamByItemName(item.StreamName.String)
		selectedStreamList = append(selectedStreamList, stream)

	} else if state == "favourites_selected" {
		fv, _ := db.GetFavouritesByItemName(item.StreamName.String)
		selectedFavList = append(selectedFavList, fv)
	} 
	if index != nil {
		currentPlaylistItemIndex = index
	} else {
		currentPlaylistItemIndex = int(selectedFavList[0].Id.Int64)
	}

	if event.Button() == 3 {
		row := wnd.PlaylistView.RowAtIndex(currentPlaylistItemIndex.(int))
		wnd.PlaylistView.SelectRow(row)	
		wnd.PlaylistView.ShowAll()
		wnd.ThreeButtonPressHandler()
	}
	return
}

func reverse(str string) (result string) {
    for _, v := range str {
        result = string(v) + result
    }
    return
}

func (wnd *MainWindow) showMetadata(forward bool) {
	metadata := getMetadata()
	var streamLogo = fmt.Sprintf("%s/./radio_logos/%s", resource_path, player.StreamLogo)
	glib.IdleAdd(func(){
		pixbuf, _ := gdkpixbuf.NewPixbufFromFileAtScale(streamLogo, 32, 32, true)
		wnd.StreamLogoView.SetFromPixbuf(pixbuf)
	})
	fmt.Println(player.playing_state)
	if metadata.Tags != nil && (player.playing_state != player.Stopped) {
		stream_title := strings.TrimSpace(metadata.Tags.Title)
		stream_caption := fmt.Sprintf("%s - %s", player.StreamName, stream_title)
		fmt.Println(stream_caption)
		num_chars := 30
		currentTitle = append(currentTitle, stream_caption)
		if len(currentTitle) == 2 {
			slices.Delete(currentTitle, 1, 2)
		}
		if len(currentTitle) == 2 && currentTitle[1] != currentTitle[0] {
			wnd.updateMetadata()
		}
		if stream_title == "" {
			wnd.MetadataView.SetText(player.StreamName)
		}
		s := len(stream_caption) - num_chars
		stream_caption = strings.ToValidUTF8(stream_caption, "")
		for {
			for j := 0; j < 30; j++ {
				time.Sleep(time.Millisecond)
				if player.playing_state == player.Playing {
					break
				}
			}
			go func() {
				if player.playing_state != player.Playing && 
						len(stream_caption) >= ind + num_chars {
					text := stream_caption[ind:ind+num_chars]
					if utf8.ValidString(text) {
						glib.IdleAdd(func() {
							wnd.MetadataView.SetText(text)
						})
					}
				}
			}()
			if forward {
				ind++
				if ind == s {
					break
				}
			} else {
				ind--
				if ind == 0 {
					break
				}
			}
		}
	} else {
		wnd.MetadataView.SetText(player.StreamName)
	}
}

func (wnd *MainWindow) updateMetadata() {	
	if player.playing_state == player.Playing {
		player.playing_state = player.MetadataUpdated
	} else {
		time.Sleep(time.Second)
	}
	if forward {
		forward = false
	} else {
		forward = true
	}
	wnd.showMetadata(forward)
	wnd.updateMetadata()
}

func setPlayerMetadata(i any) {
	if state == "favourites_selected"  {
		if i != nil {
			favouriteItem = i.(stream_db.FavouriteItem)
		}
		player.StreamName = favouriteItem.StreamName.String
		player.StreamLogo = favouriteItem.Logo.String
		player.StreamUrl = favouriteItem.Url.String
		fmt.Println("favourite item is", favouriteItem)
	} else {
		if i != nil {
			streamItem = i.(stream_db.StreamItem)
		}
		player.StreamName = streamItem.StreamName.String
		player.StreamLogo = streamItem.Logo.String
		player.StreamUrl = streamItem.Url.String
		fmt.Println("stream item is", streamItem)
	}
}

func (wnd *MainWindow) SelectedRowHandler(i any) {
	setPlayerMetadata(i)
	// if click == 0 {
	// 	player.playing_state = player.Started
	// } else {
	// 	player.playing_state = player.ItemChanged
	// }
}

func (wnd *MainWindow) ThreeButtonPressHandler() {
	var menu = gtk.NewMenu()
	menu.Attach(wnd.PlaylistView, 0, 0, 0, 0)
	var menuItem = gtk.NewMenuItemWithLabel("Add to favorites")
	menuItem.ConnectButtonPressEvent(func(event *gdk.EventButton) (ok bool) {
		if state == "default" {
			for _, stream := range selectedStreamList {
				item := stream_db.FavouriteItem{StreamName: stream.StreamName, 
					Logo: stream.Logo, Url: stream.Url}
				if !slices.Contains(favList, item) {
					db.AddToFavourites(stream.StreamName.String,
						stream.Url.String, stream.Logo.String)
				}
			}
		}
		return
	})
	var menuItem_2 = gtk.NewMenuItemWithLabel("Copy url")
	menuItem_2.ConnectButtonPressEvent(func(event *gdk.EventButton) (ok bool) {
		if len(selectedStreamList) == 1 {
			exec.Command("echo", selectedStreamList[0].Url.String).Run()
			exec.Command("xclip", "-i").Run()
		}
		return
	})
	var menuItem_3 = gtk.NewMenuItemWithLabel("Remove")
	menuItem_3.ConnectButtonPressEvent(func(event *gdk.EventButton) (ok bool) {
		if state == "favourites_selected" {
			db.RemoveFavoriteItem(wnd.PlaylistView.SelectedRow().Index())
			wnd.PlaylistView.Remove(wnd.PlaylistView.SelectedRow())
		}
		return
	})
	var menuItem_4 = gtk.NewMenuItemWithLabel("Properties")
	menuItem_4.ConnectButtonPressEvent(func(event *gdk.EventButton) (ok bool) {
		dlg := NewStreamPropertiesDialog()
		if len(selectedStreamList) > 0 {
			item := selectedStreamList[0]
			dlg.Init(item)	
		} else if len(selectedFavList) > 0 {
			item := selectedFavList[0].ToStream()
			dlg.Init(item)
		}
		
		dlg.Dialog.ShowAll()
		return
	})
	menu.Add(menuItem)
	menu.Add(menuItem_2)
	menu.Add(menuItem_3)
	menu.Add(menuItem_4)
	menu.ShowAll()
	menu.PopupAtPointer(nil)
}
