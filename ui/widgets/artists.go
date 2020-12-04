/*
 * Jellycli is a terminal music player for Jellyfin.
 * Copyright (C) 2020 Tero Vierimaa
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package widgets

import (
	"fmt"
	"github.com/gdamore/tcell"
	"gitlab.com/tslocum/cview"
	"tryffel.net/go/jellycli/config"
	"tryffel.net/go/jellycli/interfaces"
	"tryffel.net/go/jellycli/models"
	"tryffel.net/go/jellycli/util"
	"tryffel.net/go/twidgets"
)

type ArtistList struct {
	*itemList
	paging         *PageSelector
	selectFunc     func(artist *models.Artist)
	selectPageFunc func(page interfaces.Paging)
	artists        []*ArtistCover

	pagingEnabled bool
	page          interfaces.Paging
}

func NewArtistList(selectFunc func(artist *models.Artist)) *ArtistList {
	a := &ArtistList{
		itemList:   newItemList(nil),
		selectFunc: selectFunc,
		artists:    make([]*ArtistCover, 0),
	}
	a.paging = NewPageSelector(a.selectPage)

	a.list.SetInputCapture(a.listHandler)
	a.list.Grid.SetColumns(1, -1)

	a.list.Padding = 1
	a.list.ItemHeight = 2

	a.pagingEnabled = true
	selectables := []twidgets.Selectable{a.prevBtn, a.paging.Previous, a.paging.Next, a.list}
	a.Banner.Selectable = selectables

	a.Banner.Grid.SetRows(1, 1, 1, 1, -1)
	a.Banner.Grid.SetColumns(6, 2, 10, -1, 10, -1, 10, -3)
	a.Banner.Grid.SetMinSize(1, 6)

	a.Banner.Grid.AddItem(a.prevBtn, 0, 0, 1, 1, 1, 5, false)
	a.Banner.Grid.AddItem(a.description, 0, 2, 2, 6, 1, 10, false)
	a.Banner.Grid.AddItem(a.paging, 3, 4, 1, 3, 1, 10, true)
	a.Banner.Grid.AddItem(a.list, 4, 0, 1, 8, 4, 10, false)
	return a
}

func (a *ArtistList) SetText(text string) {
	a.description.SetText(text)
}

// EnablePaging enables paging and shows page on banner
func (a *ArtistList) EnablePaging(enabled bool) {
	if a.pagingEnabled && enabled {
		return
	}
	if !a.pagingEnabled && !enabled {
		return
	}
	a.pagingEnabled = enabled
	if enabled {
		selectables := []twidgets.Selectable{a.prevBtn, a.paging.Previous, a.paging.Next, a.list}
		a.Banner.Selectable = selectables
		a.Banner.Grid.AddItem(a.paging, 3, 4, 1, 3, 1, 10, true)
	} else {
		selectables := []twidgets.Selectable{a.prevBtn, a.list}
		a.Banner.Selectable = selectables
		a.Banner.Grid.RemoveItem(a.paging)
		a.page.CurrentPage = 0
	}
}

func (a *ArtistList) Clear() {
	a.list.Clear()
	a.artists = make([]*ArtistCover, 0)
}

func (a *ArtistList) SetPage(paging interfaces.Paging) {
	a.paging.SetPage(paging.CurrentPage)
	a.paging.SetTotalPages(paging.TotalPages)
	a.page = paging
}

func (a *ArtistList) AddArtists(artists []*models.Artist) {
	items := make([]twidgets.ListItem, len(artists))

	offset := 0
	if a.pagingEnabled {
		offset = a.page.Offset()
	}

	for i, v := range artists {
		cover := newArtistCover(v)
		a.artists = append(a.artists, cover)
		if v.AlbumCount > 0 {
			cover.SetText(fmt.Sprintf("%d. %s\n%d albums %s",
				offset+i+1, v.Name, v.AlbumCount, util.SecToString(v.TotalDuration)))
		} else {
			cover.SetText(fmt.Sprintf("%d. %s\n %s",
				offset+i+1, v.Name, util.SecToString(v.TotalDuration)))
		}
		items[i] = cover
	}

	a.list.AddItems(items...)
}

func (a *ArtistList) selectArtist(index int) {
	if a.selectFunc != nil {
		artist := a.artists[index]
		a.selectFunc(artist.artist)
	}
}

func (a *ArtistList) selectPage(n int) {
	if a.selectPageFunc != nil {
		a.paging.SetPage(n)
		a.page.CurrentPage = n
		a.selectPageFunc(a.page)
	}
}

func (a *ArtistList) listHandler(key *tcell.EventKey) *tcell.EventKey {
	if key.Key() == tcell.KeyEnter && a.selectFunc != nil {
		index := a.list.GetSelectedIndex()
		artist := a.artists[index]
		a.selectFunc(artist.artist)
		return nil
	}
	return key
}

type ArtistCover struct {
	*cview.TextView
	artist *models.Artist
}

func newArtistCover(artist *models.Artist) *ArtistCover {
	a := &ArtistCover{
		TextView: cview.NewTextView(),
		artist:   artist,
	}
	a.SetBackgroundColor(config.Color.Background)
	a.SetTextColor(config.Color.Text)

	a.SetText(artist.Name)

	return a
}

func (a *ArtistCover) SetSelected(s twidgets.Selection) {
	if s == twidgets.Selected {
		a.SetTextColor(config.Color.TextSelected)
		a.SetBackgroundColor(config.Color.BackgroundSelected)
	} else if s == twidgets.Deselected {
		a.SetTextColor(config.Color.Text)
		a.SetBackgroundColor(config.Color.Background)
	} else if s == twidgets.Blurred {
		a.SetBackgroundColor(config.Color.TextDisabled)
	}
}
