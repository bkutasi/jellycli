/*
 * Copyright 2020 Tero Vierimaa
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
	"tryffel.net/go/jellycli/config"
	"tryffel.net/go/jellycli/models"
	"tryffel.net/go/jellycli/util"
	"tryffel.net/go/twidgets"
)

// Queue shows a list of songs similar to album
type Queue struct {
	*twidgets.Banner
	list        *twidgets.ScrollList
	songs       []*albumSong
	listFocused bool

	playSongFunc  func(song *models.Song)
	playSongsFunc func(songs []*models.Song)

	description *tview.TextView
	prevBtn     *button
	clearBtn    *button
	prevFunc    func()
}

//NewQueue initializes new album view
func NewQueue() *Queue {
	q := &Queue{
		Banner: twidgets.NewBanner(),
		list:   twidgets.NewScrollList(nil),

		description: tview.NewTextView(),
		prevBtn:     newButton("Back"),
		clearBtn:    newButton("Clear"),
	}

	q.list.ItemHeight = 2
	q.list.Padding = 0
	q.list.SetInputCapture(q.listHandler)
	q.list.SetBorder(true)
	q.list.SetBorderColor(config.Color.Border)

	q.SetBorder(true)
	q.SetBorderColor(config.Color.Border)
	q.list.SetBackgroundColor(config.Color.Background)
	q.Grid.SetBackgroundColor(config.Color.Background)
	q.listFocused = false

	q.Banner.Grid.SetRows(1, 1, 1, 1, -1)
	q.Banner.Grid.SetColumns(6, 2, 10, -1, 10, -1, 10, -3)
	q.Banner.Grid.SetMinSize(1, 6)

	q.Banner.Grid.AddItem(q.prevBtn, 0, 0, 1, 1, 1, 5, false)
	q.Banner.Grid.AddItem(q.description, 0, 2, 2, 5, 1, 10, false)
	q.Banner.Grid.AddItem(q.clearBtn, 3, 2, 1, 1, 1, 10, true)
	q.Banner.Grid.AddItem(q.list, 4, 0, 1, 8, 4, 10, false)

	btns := []*button{q.prevBtn, q.clearBtn}
	selectables := []twidgets.Selectable{q.prevBtn, q.clearBtn, q.list}
	for _, btn := range btns {
		btn.SetLabelColor(config.Color.ButtonLabel)
		btn.SetLabelColorActivated(config.Color.ButtonLabelSelected)
		btn.SetBackgroundColor(config.Color.ButtonBackground)
		btn.SetBackgroundColorActivated(config.Color.ButtonBackgroundSelected)
	}
	q.Banner.Selectable = selectables
	q.description.SetBackgroundColor(config.Color.Background)
	q.description.SetTextColor(config.Color.Text)
	return q
}

// AddSong adds song to queue. If index is 0, add to beginning, if -1, add to end
func (q *Queue) AddSong(song *models.Song, index int) {
	var s *albumSong
	if index == -1 {
		s := newAlbumSong(song, false)
		q.songs = append(q.songs, s)
	} else if index >= 0 || index < len(q.songs)-2 {
	}
	q.list.AddItem(s)
}

// SetSongs clears current songs and sets new ones
func (q *Queue) SetSongs(songs []*models.Song) {
	q.Clear()
	q.songs = make([]*albumSong, len(songs))
	items := make([]twidgets.ListItem, len(songs))
	for i, v := range songs {
		s := newAlbumSong(v, false)
		q.songs[i] = s
		items[i] = s
	}
	q.list.AddItems(items...)
	q.printDescription()
}

// Clear removes all songs
func (q *Queue) Clear() {
	q.list.Clear()
	q.songs = []*albumSong{}
}

func (q *Queue) printDescription() {
	var text string
	if len(q.songs) == 0 {
		text = "Queue"
	} else {
		duration := 0
		for _, v := range q.songs {
			duration += v.song.Duration
		}
		text = fmt.Sprintf("Queue: %d items\n%s", len(q.songs), util.SecToStringApproximate(duration))
	}
	q.description.SetText(text)
}

func (q *Queue) listHandler(key *tcell.EventKey) *tcell.EventKey {
	if key.Key() == tcell.KeyEnter {
		//index := q.list.GetSelectedIndex()
		return nil
	}
	return key
}
