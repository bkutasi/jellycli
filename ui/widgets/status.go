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
	"sync"
	"tryffel.net/go/jellycli/config"
	"tryffel.net/go/jellycli/interfaces"
	"tryffel.net/go/jellycli/models"
	"tryffel.net/go/jellycli/util"
	"unicode/utf8"
)

const (
	btnPrevious = "|<<"
	btnPause    = " ||"
	btnStop     = "■"
	btnPlay     = ">"
	btnNext     = ">>|"
	btnForward  = ">>"
	btnBackward = " <<"
	btnQueue    = "☰"

	btnLessVolume = "\xF0\x9F\x94\x89"
	btnMoreVolume = ""

	btnStyleStart = "[white:red:b]"
	btnStyleStop  = "[-:-:-]"
)

func btn(button string) string {
	return btnStyleStart + button + btnStyleStop + " "
}

func effect(text string, e string) string {
	return fmt.Sprintf("[::%s]%s[::-]", e, text)
}

type Status struct {
	lock sync.RWMutex

	frame   *tview.Box
	layout  *tview.Grid
	details *tview.TextView

	btnPlay     *tview.Button
	btnPause    *tview.Button
	btnNext     *tview.Button
	btnPrevious *tview.Button
	btnForward  *tview.Button
	btnBackward *tview.Button
	btnStop     *tview.Button
	btnQueue    *tview.Button

	buttons   []*tview.Button
	shortCuts []string
	progress  ProgressBar
	volume    ProgressBar

	state interfaces.AudioStatus

	controlsFgColor tcell.Color
	controlsBgColor tcell.Color

	detailsFgColor tcell.Color
	detailsBgColor tcell.Color

	detailsMainColor tcell.Color
	detailsDimColor  tcell.Color

	song *models.SongInfo

	actionCb func(state interfaces.AudioStatus)

	player interfaces.Player
}

func newStatus(ctrl interfaces.Player) *Status {
	s := &Status{frame: tview.NewBox()}
	s.player = ctrl

	colors := config.Color.Status

	s.controlsBgColor = colors.ButtonBackground
	s.controlsFgColor = colors.ButtonLabel
	s.detailsMainColor = colors.Text
	s.detailsDimColor = colors.TextSecondary
	s.detailsBgColor = colors.Background

	s.frame.SetBackgroundColor(colors.Background)
	s.layout = tview.NewGrid()
	s.layout.SetBorder(true)

	s.layout.SetRows(-1, 1, 1)
	s.layout.SetColumns(-1, 5, -1)
	s.frame.SetBorder(true)
	s.frame.SetBorderAttributes(tcell.AttrBold)
	s.frame.SetBorderColor(colors.Border)

	s.details = tview.NewTextView()

	s.btnPlay = tview.NewButton(btnPlay)
	s.btnPlay.SetSelectedFunc(s.namedCbFunc(btnPlay))
	s.btnPause = tview.NewButton(btnPause)
	s.btnPause.SetSelectedFunc(s.namedCbFunc(btnPause))
	s.btnNext = tview.NewButton(btnNext)
	s.btnNext.SetSelectedFunc(s.namedCbFunc(btnNext))
	s.btnPrevious = tview.NewButton(btnPrevious)
	s.btnPrevious.SetSelectedFunc(s.namedCbFunc(btnPrevious))
	s.btnForward = tview.NewButton(btnForward)
	s.btnForward.SetSelectedFunc(s.namedCbFunc(btnForward))
	s.btnBackward = tview.NewButton(btnBackward)
	s.btnBackward.SetSelectedFunc(s.namedCbFunc(btnBackward))
	s.btnStop = tview.NewButton(btnStop)
	s.btnStop.SetSelectedFunc(s.namedCbFunc(btnStop))
	s.btnQueue = tview.NewButton(btnQueue)

	s.progress = NewProgressBar(40, 100)
	s.volume = NewProgressBar(10, 100)

	state := interfaces.AudioStatus{
		State:    interfaces.AudioStateStopped,
		Song:     nil,
		Artist:   nil,
		Album:    nil,
		SongPast: 0,
		Volume:   50,
		Muted:    false,
	}

	s.state = state

	s.buttons = []*tview.Button{
		s.btnPrevious, s.btnBackward, s.btnPlay, s.btnForward, s.btnNext,
	}

	s.shortCuts = []string{
		util.PackKeyBindingName(config.KeyBinds.Global.Previous, 5),
		util.PackKeyBindingName(config.KeyBinds.Global.Backward, 5),
		util.PackKeyBindingName(config.KeyBinds.Global.PlayPause, 5),
		util.PackKeyBindingName(config.KeyBinds.Global.Forward, 5),
		util.PackKeyBindingName(config.KeyBinds.Global.Next, 5),
	}

	for _, v := range s.buttons {
		v.SetBackgroundColor(s.controlsBgColor)
		v.SetLabelColor(s.controlsFgColor)
		v.SetBorder(false)
	}
	return s
}

func (s *Status) Draw(screen tcell.Screen) {
	// TODO: Make drawing responsive
	s.frame.Draw(screen)
	x, y, w, _ := s.frame.GetInnerRect()

	songPast := util.SecToString(s.state.SongPast.Seconds())
	songPast = " " + songPast + " "
	var songDuration = " 0:00 "
	if s.state.Song != nil {
		songDuration = util.SecToString(s.state.Song.Duration)
		songDuration = " " + songDuration + " "
	}

	volume := " Volume " + s.volume.Draw(int(s.state.Volume))
	topRowFree := w - len(songPast) - len(songDuration) - utf8.RuneCountInString(volume) - 5

	s.progress.SetWidth(topRowFree * 10 / 11)

	s.lock.RLock()
	defer s.lock.RUnlock()

	progressBar := s.progress.Draw(s.state.SongPast.Seconds())

	progress := songPast + progressBar + songDuration
	progressLen := utf8.RuneCountInString(progress)

	topX := x + 1

	colors := config.Color.Status

	tview.Print(screen, progress, topX, y-1, progressLen+5, tview.AlignLeft, colors.ProgressBar)
	topX += progressLen + progressLen/10
	tview.Print(screen, volume, topX, y-1, w, tview.AlignLeft, colors.ProgressBar)

	tview.Print(screen, util.PackKeyBindingName(config.KeyBinds.Global.VolumeDown, 5),
		topX+7, y, topX+16, tview.AlignLeft, colors.Shortcuts)
	tview.Print(screen, util.PackKeyBindingName(config.KeyBinds.Global.VolumeUp, 5),
		topX+18, y, topX+1, tview.AlignLeft, colors.Shortcuts)

	btnY := y + 1
	btnX := x + 1

	for i, v := range s.buttons {
		tview.Print(screen, s.shortCuts[i], btnX, btnY-1, 4, tview.AlignLeft, colors.Shortcuts)

		v.SetRect(btnX, btnY, 3, 1)
		v.Draw(screen)
		btnX += 5
	}
	s.WriteStatus(screen, x+27, y)
}

func (s *Status) GetRect() (int, int, int, int) {
	return s.frame.GetRect()
}

func (s *Status) SetRect(x, y, width, height int) {
	s.frame.SetRect(x, y, width, height)
}

func (s *Status) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return s.frame.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		//key := event.Key()

	})
}

func (s *Status) Focus(delegate func(p tview.Primitive)) {
	s.frame.Focus(delegate)
}

func (s *Status) Blur() {
	s.frame.Blur()
}

func (s *Status) GetFocusable() tview.Focusable {
	return s.frame.GetFocusable()
}

func (s *Status) WriteStatus(screen tcell.Screen, x, y int) {
	if s.state.State != interfaces.AudioStateStopped &&
		(s.state.Song != nil && s.state.Album != nil && s.state.Artist != nil) {
		xi := x
		w, _ := screen.Size()
		tview.Print(screen, effect(s.state.Song.Name, "b")+" - ", x, y, w, tview.AlignLeft, s.detailsMainColor)
		x += len(s.state.Song.Name) + 3
		tview.Print(screen, effect(s.state.Artist.Name, "b")+" ", x, y, w, tview.AlignLeft, s.detailsMainColor)
		x += len(s.state.Artist.Name) + 1
		x = xi + 4
		tview.Print(screen, s.state.Album.Name+" ", x, y+1, w, tview.AlignLeft, s.detailsMainColor)
		x += len(s.state.Album.Name) + 1
		tview.Print(screen, fmt.Sprintf("(%d)", s.state.Album.Year), x, y+1, w, tview.AlignLeft, s.detailsMainColor)
	}
}

// Return cb func that includes name
func (s *Status) namedCbFunc(name string) func() {
	return func() {
		s.buttonCb(name)
	}
}

// Button pressed cb with name of the button
func (s *Status) buttonCb(name string) {
	logrus.Infof("Button %s was pressed!", name)
	if s.actionCb == nil {
		return
	}

	status := interfaces.AudioStatus{}

	switch name {
	case btnPlay:
		status.Action = interfaces.AudioActionPlay
	case btnPause:
		status.Action = interfaces.AudioActionPlayPause
	case btnNext:
		status.Action = interfaces.AudioActionNext
	}
	s.actionCb(status)
}

func (s *Status) UpdateState(state interfaces.AudioStatus, song *models.SongInfo) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.song = song
	if state.Song != nil {
		s.progress.SetMaximum(state.Song.Duration)
	}
	s.state = state
	s.DrawButtons()
}

func (s *Status) DrawButtons() {
	if s.state.Paused {
		s.btnPlay.SetLabel(btnPlay)
	} else {
		s.btnPlay.SetLabel(btnPause)
	}
}
