/*
 * Copyright 2019 Tero Vierimaa
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package widgets

import (
	"fmt"
	"github.com/gdamore/tcell"
	"github.com/rivo/tview"
	"github.com/sirupsen/logrus"
	"tryffel.net/go/jellycli/config"
	"tryffel.net/go/jellycli/interfaces"
	"tryffel.net/go/jellycli/models"
	"tryffel.net/go/jellycli/ui/widgets/modal"
	"tryffel.net/go/twidgets"
)

type Window struct {
	app    *tview.Application
	layout *twidgets.ModalLayout

	// Widgets
	navBar   *twidgets.NavBar
	status   *Status
	mediaNav *MediaNavigation
	help     *modal.Help
	queue    *Queue
	history  *Queue

	artist     *ArtistView
	album      *AlbumView
	artistList *ArtistList
	playlists  *Playlists
	playlist   *PlaylistView
	songs      *SongList

	gridAxisX  []int
	gridAxisY  []int
	customGrid bool
	modal      modal.Modal

	mediaController   interfaces.MediaController
	mediaView         Previous
	mediaViewSelected bool

	hasModal  bool
	lastFocus tview.Primitive
}

func NewWindow(mc interfaces.MediaController) Window {
	w := Window{
		app:    tview.NewApplication(),
		status: newStatus(mc),
		layout: twidgets.NewModalLayout(),
	}

	w.artistList = NewArtistList(w.selectArtist)
	w.artistList.SetBackCallback(w.goBack)
	w.artistList.selectPageFunc = w.showArtistPage
	w.artist = NewArtistView(w.selectAlbum)
	w.artist.SetBackCallback(w.goBack)
	w.album = NewAlbumview(w.playSong, w.playSongs)
	w.album.SetBackCallback(w.goBack)
	w.mediaNav = NewMediaNavigation(w.selectMedia)
	w.navBar = twidgets.NewNavBar(config.Color.NavBar.ToWidgetsNavBar(), w.navBarHandler)

	w.playlists = NewPlaylists(w.selectPlaylist)
	w.playlist = NewPlaylistView(w.playSong, w.playSongs)
	w.playlist.SetBackCallback(w.goBack)

	w.songs = NewSongList(w.playSong, w.playSongs)
	w.songs.SetBackCallback(w.goBack)
	w.songs.showPage = w.selectSongs
	w.mediaController = mc

	w.setLayout()
	w.app.SetRoot(w.layout, true)
	w.app.SetFocus(w.mediaNav)

	w.app.SetInputCapture(w.eventHandler)
	//w.window.SetInputCapture(w.eventHandler)
	w.help = modal.NewHelp(w.closeHelp)
	w.help.SetDoneFunc(w.wrapCloseModal(w.help))
	w.queue = NewQueue()
	w.queue.SetBackCallback(w.goBack)
	w.mediaController.SetQueueChangedCallback(func(songs []*models.Song) {
		w.app.QueueUpdate(func() {
			w.queue.SetSongs(songs)
		})
	})

	w.history = NewQueue()
	w.history.SetHistoryMode(true)
	w.history.SetBackCallback(w.goBack)

	w.mediaController.SetHistoryChangedCallback(func(songs []*models.Song) {
		w.app.QueueUpdate(func() {
			w.history.SetSongs(songs)
		})
	})

	w.layout.Grid().SetBackgroundColor(config.Color.Background)

	w.mediaController.AddStatusCallback(w.statusCb)

	navBarLabels := []string{"Help", "Queue", "History"}

	sc := config.KeyBinds.NavigationBar
	navBarShortucts := []tcell.Key{sc.Help, sc.Queue, sc.History}

	for i, v := range navBarLabels {
		btn := tview.NewButton(v)
		w.navBar.AddButton(btn, navBarShortucts[i])
	}

	return w
}

func (w *Window) Run() error {
	return w.app.Run()
}

func (w *Window) Stop() {
	w.app.Stop()
}

func (w *Window) setLayout() {
	w.gridAxisY = []int{1, -1, -2, -2, -1, 4}
	w.gridAxisX = []int{24, -1, -2, -2, -1, 24}

	w.layout.SetGridXSize([]int{10, -1, -1, -1, -1, -1, -1, -1, -1, 10})
	w.layout.SetGridYSize([]int{1, -1, -1, -1, -1, -1, -1, -1, -1, 5})

	w.layout.Grid().AddItem(w.navBar, 0, 0, 1, 10, 1, 30, false)
	w.layout.Grid().AddItem(w.mediaNav, 1, 0, 8, 2, 5, 10, false)
	w.layout.Grid().AddItem(w.status, 9, 0, 1, 10, 3, 10, false)

	//w.setViewWidget(w.artistList)
}

// go back to previous primitive
func (w *Window) goBack(p Previous) {
	w.setViewWidget(p, false)
}

// set central widget. If updatePrevious, set update previous primitive's last primitive
func (w *Window) setViewWidget(p Previous, updatePrevious bool) {
	if p == w.mediaView {
		return
	}

	last := w.mediaView
	w.lastFocus = w.app.GetFocus()
	w.layout.Grid().RemoveItem(w.mediaView)
	w.layout.Grid().AddItem(p, 1, 2, 8, 8, 15, 10, false)
	w.app.SetFocus(p)
	w.mediaView = p
	if updatePrevious {
		p.SetLast(last)
	}
}

func (w *Window) eventHandler(event *tcell.EventKey) *tcell.EventKey {

	out := w.keyHandler(event)
	if out == nil {
		return nil
	}
	return event
}

func (w *Window) navBarHandler(label string) {

}

// Key handler, if match, return nil
func (w *Window) keyHandler(event *tcell.EventKey) *tcell.Key {

	key := event.Key()
	/*
		if key >= tcell.KeyF1 && key <= tcell.KeyF12 && !w.navBarFocused{
			//Activate navigation bar on function button
			w.lastFocus = w.app.GetFocus()
			w.lastFocus.Blur()
			w.app.SetFocus(w.navBar)
			w.navBarFocused = true
		} else if key == tcell.KeyEscape && w.navBarFocused {
			//Deactivate navigation bar and return to last focus
			w.navBarFocused = false
			w.navBar.Blur()
			w.app.SetFocus(w.lastFocus)
			w.lastFocus = nil
			return nil
		}
	*/

	if w.mediaCtrl(event) {
		return nil
	}
	if w.navBarCtrl(key) {
		return nil
	}
	if w.moveCtrl(key) {
		return nil
	}
	// Moving around
	return &key
}

func (w *Window) mediaCtrl(event *tcell.EventKey) bool {
	ctrls := config.KeyBinds.Global
	key := event.Key()
	switch key {
	case ctrls.PlayPause:
		if w.status.state.State == interfaces.Pause {
			go w.mediaController.Continue()
		} else if w.status.state.State == interfaces.Play {
			go w.mediaController.Pause()
		}
	case ctrls.VolumeDown:
		volume := w.status.state.Volume - 5
		go w.mediaController.SetVolume(volume)
	case ctrls.VolumeUp:
		volume := w.status.state.Volume + 5
		go w.mediaController.SetVolume(volume)
	case ctrls.Next:
		w.mediaController.Next()
	default:
		return false
	}
	//w.status.InputHandler()(event, nil)
	return true
}

func (w *Window) navBarCtrl(key tcell.Key) bool {
	navBar := config.KeyBinds.NavigationBar
	switch key {
	// Navigation bar
	case navBar.Quit:
		w.app.Stop()
	case navBar.Help:
		stats := w.mediaController.GetStatistics()
		w.help.SetStats(stats)
		w.showModal(w.help, 25, 50, true)
	case navBar.Queue:
		w.setViewWidget(w.queue, true)
	case navBar.History:
		w.setViewWidget(w.history, true)
		items := w.mediaController.GetHistory(100)
		duration := 0
		for _, v := range items {
			duration += v.Duration
		}
	default:
		return false
	}
	return true
}

func (w *Window) moveCtrl(key tcell.Key) bool {
	if key == tcell.KeyTAB {
		if w.mediaViewSelected {
			w.lastFocus = w.mediaView
			w.app.SetFocus(w.mediaNav)
			if w.lastFocus != nil {
				w.lastFocus.Blur()
			}
			w.mediaViewSelected = false
		} else {
			w.lastFocus = w.app.GetFocus()
			w.app.SetFocus(w.mediaView)
			w.mediaViewSelected = true
			if w.lastFocus != nil {
				w.lastFocus.Blur()
			}
		}
		return true
	}
	return false
}

func (w *Window) searchCb(query string, doSearch bool) {
	logrus.Debug("In search callback")
	w.app.SetFocus(w.layout)

	if doSearch {
		//w.mediaController.Search(query)
	}

}

func (w *Window) closeHelp() {
	w.app.SetFocus(w.layout)
}

func (w *Window) wrapCloseModal(modal modal.Modal) func() {
	return func() {
		w.closeModal(modal)
	}
}

func (w *Window) closeModal(modal modal.Modal) {
	if w.hasModal {
		modal.Blur()
		modal.SetVisible(false)
		w.layout.RemoveModal(modal)

		w.hasModal = false
		w.modal = nil
		w.app.SetFocus(w.lastFocus)
		w.lastFocus = nil
		w.hasModal = false
	} else {
		logrus.Warning("Trying to close modal when there's no open modal.")
	}
}

func (w *Window) showModal(modal modal.Modal, height, width uint, lockSize bool) {
	if !w.hasModal {
		w.hasModal = true
		w.modal = modal
		w.lastFocus = w.app.GetFocus()
		w.lastFocus.Blur()
		if !lockSize {
			w.layout.AddFixedModal(modal, height, width, twidgets.ModalSizeMedium)
		} else {
			w.layout.AddDynamicModal(modal, twidgets.ModalSizeLarge)
		}
		w.app.SetFocus(modal)
		modal.SetVisible(true)
		w.app.QueueUpdateDraw(func() {})
	} else {
		logrus.Warning("Trying show close modal when there's another modal open.")
	}
}

func (w *Window) statusCb(state interfaces.PlayingState) {
	w.status.UpdateState(state, nil)
	w.app.QueueUpdateDraw(func() {})
}

func (w *Window) InitBrowser(items []models.Item) {
	//w.browser.setData(items)
	w.app.Draw()
}

func (w *Window) selectMedia(m MediaSelect) {
	switch m {
	case MediaLatestMusic:
		albums, err := w.mediaController.GetLatestAlbums()
		if err != nil {
			logrus.Errorf("get favorite artists: %v", err)
		} else {
			duration := 0
			for _, v := range albums {
				duration += v.Duration
			}
			// set pseudo artist
			artist := &models.Artist{
				Id:            "",
				Name:          "Latest albums",
				Albums:        nil,
				TotalDuration: duration,
				AlbumCount:    len(albums),
			}

			w.mediaNav.SetCount(MediaLatestMusic, len(albums))
			w.artist.Clear()
			w.artist.SetArtist(artist)
			w.artist.SetAlbums(albums)
			w.setViewWidget(w.artist, true)
		}
	case MediaFavoriteArtists:
		artists, err := w.mediaController.GetFavoriteArtists()
		if err != nil {
			logrus.Errorf("get favorite artists: %v", err)
		} else {
			w.artistList.Clear()
			w.artistList.SetText("Favorite artists")
			w.artistList.EnablePaging(false)
			w.mediaNav.SetCount(MediaFavoriteArtists, len(artists))
			w.artistList.AddArtists(artists)
			w.setViewWidget(w.artistList, true)
		}
	case MediaPlaylists:
		playlists, err := w.mediaController.GetPlaylists()
		if err != nil {
			logrus.Errorf("get playlists: %v", err)
		} else {
			w.mediaNav.SetCount(MediaPlaylists, len(playlists))
			w.playlists.SetPlaylists(playlists)
			w.setViewWidget(w.playlists, true)
		}
	case MediaSongs:
		paging, err := w.mediaController.GetAllSongsCount()
		if err != nil {
			logrus.Errorf("get songs count: %v", err)
			return
		}

		songs, err := w.mediaController.GetSongs(0, paging.PageSize)
		if err != nil {
			logrus.Errorf("get songs: %v", err)
		}

		w.mediaNav.SetCount(MediaSongs, paging.TotalItems)
		w.songs.SetSongs(songs, paging)

		w.setViewWidget(w.songs, true)
	case MediaArtists:
		paging := interfaces.Paging{
			CurrentPage: 0,
			PageSize:    100,
		}
		artists, total, err := w.mediaController.GetArtists(paging)
		if err != nil {
			logrus.Errorf("get all artists: %v", err)
			return
		}
		paging.SetTotalItems(total)
		w.mediaNav.SetCount(MediaArtists, total)

		w.artistList.Clear()
		w.artistList.EnablePaging(true)
		w.artistList.SetPage(paging)

		w.artistList.AddArtists(artists)
		w.setViewWidget(w.artistList, true)
		w.artistList.SetText(fmt.Sprintf("All artists: %d", paging.TotalItems))
	}
}

func (w *Window) selectArtist(artist *models.Artist) {
	albums, err := w.mediaController.GetArtistAlbums(artist.Id)
	if err != nil {
		logrus.Errorf("get artist albums: %v", err)
	} else {
		artist.AlbumCount = len(albums)
		w.artist.SetArtist(artist)
		w.artist.SetAlbums(albums)
		w.setViewWidget(w.artist, true)
	}
}

func (w *Window) selectAlbum(album *models.Album) {
	songs, err := w.mediaController.GetAlbumSongs(album.Id)
	if err != nil {
		logrus.Errorf("get album songs: %v", err)
	} else {
		for _, v := range songs {
			v.AlbumArtist = album.Artist
		}

		w.album.SetAlbum(album, songs)
		w.album.SetLast(w.mediaView)
		w.setViewWidget(w.album, true)
	}
}

func (w *Window) selectPlaylist(playlist *models.Playlist) {
	err := w.mediaController.GetPlaylistSongs(playlist)
	if err != nil {
		logrus.Warningf("did not get playlist songs: %v", err)
		return
	}

	w.playlist.SetPlaylist(playlist)
	w.setViewWidget(w.playlist, true)
}

func (w *Window) selectSongs(page interfaces.Paging) {
	songs, err := w.mediaController.GetSongs(page.CurrentPage, page.PageSize)
	if err != nil {
		logrus.Errorf("get songs: %v", err)
	}

	w.songs.SetSongs(songs, page)
	w.setViewWidget(w.songs, true)
}

func (w *Window) showArtistPage(page interfaces.Paging) {
	artists, _, err := w.mediaController.GetArtists(page)
	if err != nil {
		logrus.Errorf("get artist page: %v", err)
		return
	}

	w.artistList.Clear()
	w.artistList.AddArtists(artists)
	w.artistList.EnablePaging(true)
	w.setViewWidget(w.artistList, false)
}

func (w *Window) playSong(song *models.Song) {
	w.playSongs([]*models.Song{song})
}

func (w *Window) playSongs(songs []*models.Song) {
	w.mediaController.AddSongs(songs)
}
