package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"strings"

	// "sync"
	"time"

	// "unicode/utf16"
	// "unicode/utf8"

	"github.com/atotto/clipboard"
	"github.com/diamondburned/gotk4/pkg/gdk/v3"
	"github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v3"
	"github.com/gen2brain/dlgs"
	"golang.org/x/exp/slices"
	"gopkg.in/vansante/go-ffprobe.v2"

	stream_db "github.com/peter7rom-star/im-player/db"
)

var homeDir, _ = os.UserHomeDir()
var click, ind, s int
var builder = gtk.NewBuilder()
var player = NewPlayer()
var wnd *MainWindow
var dlg *StreamPropertiesDialog
var resource_path = fmt.Sprintf("%s/.local/share/ims-player", homeDir)
var settingsFile = fmt.Sprintf("%s/settings.json", resource_path)
var db = stream_db.InitDB(fmt.Sprintf("%s/db/metadata.db", resource_path))
var state, playlistState, playlistViewState, DefaultViewState, query, query_2 string
var forward = false
var currentPlaylistItemIndex interface{}
var selectedStreamList []stream_db.StreamItem
var favList []stream_db.FavouriteItem
var streamItem stream_db.StreamItem
var favouriteItem stream_db.FavouriteItem
var landList, currentTitle []string


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
	wnd.SelectCountryBox.SetWrapWidth(10)
	query = wnd.SelectCountryBox.ActiveText()
	player.StreamList = db.LoadStationList(nil)
	var data []stream_db.StreamItem
	favList = db.LoadFavourites()
	input, err := os.ReadFile(settingsFile)
	var count int
	if err == nil {
		var settingsData SettingsData
		err := json.Unmarshal(input, &settingsData)
		if err == nil {
			DefaultViewState = settingsData.DefaultViewState
			fmt.Println(DefaultViewState)
			if DefaultViewState == "All stations" {
				data = player.StreamList
				state = "default"
			} else {
				for _, item := range favList {
					data = append(data, item.ToStream())
				}
				state = "favourites_selected"
			}
		} else {
			log.Fatal(err)
		}
	} else {
		data = player.StreamList
		log.Fatal(err)
	}
	wnd.PlaylistView = gtk.NewListBox()
	listView_2 := gtk.NewListBox()
	listView_3 := gtk.NewListBox()
	// var m sync.Mutex
	for ind, item := range data {
		wnd.AddItemToPlaylist(ind, item)
	}
	landList = db.LoadLandList()
	model := append(landList, "favourites")
	for _, elem := range model {
		wnd.SelectCountryBox.AppendText(elem)
	}
	wnd.SelectCountryBox.SetActive(0)
	if DefaultViewState == "Favourites" {
		wnd.SelectCountryBox.SetActive(len(model))
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
				// go wnd.updateMetadata()
			}()
		} else {
			// m.Lock()
			go func() {
				player.StopPlayback()
				player.playing_state = player.Playing
				player.Play()
			}()
			// m.Unlock()
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
	wnd.PrefsButton.ConnectClicked(func() {
		dlg := NewSettingsDialog()
		dlg.Init()
		dlg.Dialog.ShowAll()
	})
	wnd.SelectCountryBox.SetCanFocus(true)
	wnd.FavButton.ConnectClicked(func() {
		count++
		wnd.PlaylistView = listView_2
		state = "favourites_selected"
		ind := slices.Index(model, "favourites")
		ind++
		wnd.SelectCountryBox.SetActive(ind)
	})
	wnd.LibButton.ConnectClicked(func() {
		count++
		state = "library_selected"
		ind := slices.Index(model, query_2)
		ind++
		if DefaultViewState == "Favourites" {
			ind = 0
		}
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
		state = "default"
		selectedStreamList = []stream_db.StreamItem{}
		selectedFavList = []stream_db.FavouriteItem{}
		if query == "All" || len(query) == 0 {
			wnd.PlaylistView = gtk.NewListBox()
			if DefaultViewState == "Favourites" && count == 1 {
				for _, item := range player.StreamList {
					wnd.AddItemToPlaylist(nil, item)
				}
				listView = wnd.PlaylistView
			} else {
				wnd.PlaylistView = listView
				query_2 = query
			}
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
		count++
		// if DefaultViewState == "Favourites" && count == 1 {
		// 	wnd.PlaylistView = gtk.NewListBox()
		// 	for _, item := range player.StreamList {
		// 		wnd.AddItemToPlaylist(nil, item)
		// 	}
		// 	listView = wnd.PlaylistView
		// }
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

func (dlg *StreamPropertiesDialog) Init(index any, item stream_db.StreamItem, nameTypeOfItem string) {
	playlistIndex := index
	oldStreamName := item.StreamName.String
	dlg.StreamNameBox.SetText(item.StreamName.String)
	dlg.StreamUrlBox.SetText(item.Url.String)
	var bitrate string
	metadata, err := getMetadata()
	if err != nil {
		bitrate = "undefined"
	} else {
		btr := metadata.BitRate
		if len(btr) > 0 {
			bitrate = btr[:3]
			bitrate = fmt.Sprintf("%s %s", bitrate, "kb/s")
		}
	}
	dlg.StreamBitrateBox.SetText(bitrate)
	dlg.StreamCountryBox.SetText(item.Country.String)	
	dlg.OkButton.ConnectReleased(func() {
		db.Update(oldStreamName, dlg.StreamNameBox.
			Text(), dlg.StreamUrlBox.Text())
		dlg.Dialog.Hide()

		if nameTypeOfItem == "StreamItem" {
			var data []stream_db.StreamItem
			if slices.Contains(landList, query) {
				data = db.LoadStationListFromCountry(query)
			} else {
				data = db.LoadStationList(query)
			}

			for ind, item := range data {
				if playlistIndex != nil && ind == playlistIndex {
					row := wnd.PlaylistView.RowAtIndex(ind)
					wnd.PlaylistView.Remove(row)
					newRow, _ := addRow(item)
					wnd.PlaylistView.Insert(newRow, ind)
				}   
			}
		} else {
			var data = db.LoadFavourites()
			for ind, item := range data {
				if playlistIndex != nil && ind == playlistIndex {
					convItem := item.ToStream()
					row := wnd.PlaylistView.RowAtIndex(ind)
					wnd.PlaylistView.Remove(row)
					newRow, _ := addRow(convItem)
					wnd.PlaylistView.Insert(newRow, ind)
				}
			}
		}
		wnd.PlaylistView.ShowAll()
		wnd.ViewPort.Add(wnd.PlaylistView)
	})
	
	dlg.CancelButton.ConnectReleased(func ()  {
		dlg.Dialog.Hide()
	})
	dlg.Dialog.ConnectClose(dlg.Dialog.Hide)
}

func NewSettingsDialog() *SettingsDialog {
	dialog := builder.GetObject("settings_dialog").Cast().(*gtk.Dialog)
	// interfaceBox := builder.GetObject("interface_box").Cast().(*gtk.ComboBoxText)
	defaultViewBox := builder.GetObject("default_view_box").Cast().(*gtk.ComboBoxText)
	okButton := builder.GetObject("ok_butto").Cast().(*gtk.Button)
	cancelButton := builder.GetObject("cancel_butto").Cast().(*gtk.Button)
	return &SettingsDialog{Dialog: dialog, DefaultViewBox: defaultViewBox, 
						   OkButton: okButton, CancelButton: cancelButton,}
}

func (dlg *SettingsDialog) Init() {
	defaultViewState := dlg.DefaultViewBox.ActiveText()
	dlg.DefaultViewBox.ConnectChanged(func() {
		defaultViewState = dlg.DefaultViewBox.ActiveText()
	})
	dlg.OkButton.ConnectClicked(func() {
		data := SettingsData{DefaultViewState: defaultViewState}
		output, err := json.Marshal(data)
		if err == nil {
			err = os.WriteFile(settingsFile, output, os.ModePerm)
			if err != nil {
				log.Fatal(err)
			}
		} else {
			log.Fatal(err)
		}
		dlg.Dialog.Close()
	})
	dlg.CancelButton.ConnectClicked(func() {
		dlg.Dialog.Close()
	})

}

func main() {
	gtk.Init()
	wnd = NewMainWindow()
	app := gtk.NewApplication("com.github.peter7rom-star.ims-player", 0)
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

var selectedFavList []stream_db.FavouriteItem

func addRow(item stream_db.StreamItem) (*gtk.ListBoxRow, stream_db.StreamItem) {
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
	return row, item
}

func (wnd *MainWindow) AddItemToPlaylist(index any, item stream_db.StreamItem) {
	row, item := addRow(item)
	wnd.PlaylistView.Add(row)
	row.ConnectButtonPressEvent(func(event *gdk.EventButton) (ok bool) {
		wnd.onRowClickHandler(index, item, event)
		return
	})
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
		wnd.ThreeButtonPressHandler(index)
	}
	return
}

func getMetadata() (*ffprobe.Format, error) {
	var metadata *ffprobe.Format
	var err error
	metadata_ch := make(chan *ffprobe.Format, 500)
	error_ch := make(chan error)
	go player.GetStreamMetadata(metadata_ch, error_ch)
	select {
	case metadata = <- metadata_ch:
		return metadata, err
	case err = <- error_ch:
		return metadata, err
	}
}

var stream_title, stream_tag string

func (wnd *MainWindow) showMetadata(forward bool) {
	metadata, err := getMetadata()
	var streamLogo = fmt.Sprintf("%s/./radio_logos/%s", resource_path, player.StreamLogo)
	glib.IdleAdd(func(){
		pixbuf, _ := gdkpixbuf.NewPixbufFromFileAtScale(streamLogo, 32, 32, true)
		wnd.StreamLogoView.SetFromPixbuf(pixbuf)
	})
	fmt.Println(player.playing_state)
	if player.playing_state != player.Stopped {
		if err != nil {
			wnd.MetadataView.SetText(player.StreamTitle)
			ind = 0
			forward = true
			player.playing_state = player.Playing
			wnd.updateMetadata()
		} 
		stream_tag, err = metadata.TagList.GetString("StreamTitle")
		if stream_tag != "" || err == nil {
			stream_tag = strings.TrimSpace(stream_tag)
			stream_title = fmt.Sprintf("%s - %s", player.StreamTitle, stream_tag)
		} else {
			wnd.MetadataView.SetText(player.StreamTitle)
			return
		}	
		num_chars := 30
		currentTitle = append(currentTitle, stream_title)
		fmt.Println("length of string stream tag is", len(stream_tag))
		fmt.Println(stream_title)
		if stream_tag != "" {
			if len(currentTitle) == 2 {
				slices.Delete(currentTitle, 1, 2)
			}
			if len(currentTitle) == 2 && currentTitle[1] != currentTitle[0] {
				ind = 0
				forward = false
				player.playing_state = player.Playing
			}
		}
		stream_title = strings.ToValidUTF8(stream_title, "")
		s := len(stream_title) - num_chars + 1
		var text string
		for {
			for j := 0; j < 30; j++ {
				time.Sleep(time.Millisecond)
				if player.playing_state == player.Playing {
					wnd.updateMetadata()
				} 
			}
			l := ind + num_chars - 1
			// fmt.Println(ind, s)
			go func() {
				if player.playing_state != player.Playing {	
					if ind >= 0 {
						if len(stream_title) >= l {
							text = stream_title[ind:l]
						}
						// } else if ind == s - 1 {
						// 	text = stream_title[ind:num_chars-1]
						// }
						if len(stream_tag) == 0 { 
							if ind >= s && s > 0 {
								return
							} else {
								text = stream_title
							}
						}
					}
				}
				glib.IdleAdd(func() {
					wnd.MetadataView.SetTextWithMnemonic(text)
				})
				fmt.Println(ind)
			}()
			if forward {
				if ind >= s {
					break
				}
				ind++
			} else {
				if ind <= 0 {
					break
				}
				ind--
			}
		}
	}
}

func (wnd *MainWindow) updateMetadata() {	
	if player.playing_state == player.Playing {
		player.playing_state = player.MetadataUpdated
		forward = true
		ind = 0
	} else {
		if forward {
			forward = false
		} else {
			forward = true
		}
		time.Sleep(time.Second)
	}
	wnd.showMetadata(forward)
	wnd.updateMetadata()
}

func setPlayerMetadata(i any) {
	if state == "favourites_selected"  {
		if i != nil {
			favouriteItem = i.(stream_db.FavouriteItem)
		}
		player.StreamTitle = favouriteItem.StreamName.String
		player.StreamLogo = favouriteItem.Logo.String
		player.StreamUrl = favouriteItem.Url.String
		fmt.Printf("stream name is %s", player.StreamTitle)
	} else {
		if i != nil {
			streamItem = i.(stream_db.StreamItem)
		}
		player.StreamTitle = streamItem.StreamName.String
		player.StreamLogo = streamItem.Logo.String
		player.StreamUrl = streamItem.Url.String
		fmt.Printf("stream name is %s", player.StreamTitle)
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

func (wnd *MainWindow) ThreeButtonPressHandler(index any) {
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
			fmt.Println(selectedStreamList[0].Url.String)
			clipboard.WriteAll(selectedStreamList[0].Url.String)
		} else if len(selectedFavList) == 1 {
			fmt.Println(selectedFavList[0].Url.String)
			clipboard.WriteAll(selectedFavList[0].Url.String)
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
		dlg = NewStreamPropertiesDialog()
		var item stream_db.StreamItem
		var nameTypeOfItem string
		if len(selectedStreamList) > 0 {
			item = selectedStreamList[0]
			go wnd.SelectedRowHandler(item)
			nameTypeOfItem = reflect.TypeOf(item).Name()
		} else if len(selectedFavList) > 0 {
			item = selectedFavList[0].ToStream()
			go wnd.SelectedRowHandler(selectedFavList[0])
			nameTypeOfItem = reflect.TypeOf(selectedFavList[0]).Name()
		}
		dlg.Init(index, item, nameTypeOfItem)
		
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
