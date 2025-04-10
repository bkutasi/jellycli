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

// Package player contains all logic for jellycli. This includes queue (history) management, low-level audio and
// audio controls.
package player

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"strings"
	"sync"
	"time"
	"tryffel.net/go/jellycli/api"
	"tryffel.net/go/jellycli/config"
	"tryffel.net/go/jellycli/interfaces"
	"tryffel.net/go/jellycli/models"
	"tryffel.net/go/jellycli/task"
)

type songMetadata struct {
	song          *models.Song
	album         *models.Album
	artist        *models.Artist
	albumImageUrl string
	albumImageId  string
	reader        io.ReadCloser
	format        interfaces.AudioFormat
}

// Player wraps all controllers and implements interfaces.QueueController, interfaces.Player and
// interfaces.ItemController.
type Player struct {
	task.Task
	*Audio
	*Queue

	lock *sync.RWMutex

	downloadingSong bool

	songComplete   chan bool
	audioUpdated   chan models.AudioStatus
	songDownloaded chan songMetadata

	nextSong *songMetadata

	api              interfaces.Api // Use the interface from the interfaces package
	remoteController api.RemoteController

	lastApiReport time.Time
}

// initialize new player. This also initializes faiface.Speaker, which should be initialized only once.
func NewPlayer(browser interfaces.Api) (*Player, error) { // Use the interface from the interfaces package
	var err error
	p := &Player{
		lock:           &sync.RWMutex{},
		songComplete:   make(chan bool, 3),
		audioUpdated:   make(chan models.AudioStatus, 3),
		songDownloaded: make(chan songMetadata, 3),
		api:            browser,
	}
	p.Name = "Player"
	p.Task.SetLoop(p.loop)

	p.Audio = newAudio()
	p.Queue = newQueue()
	if remoteController, ok := browser.(api.RemoteController); ok {
		p.remoteController = remoteController
		p.remoteController.SetPlayer(p)
	}

	err = initAudio()
	if err != nil {
		return p, fmt.Errorf("init audio backend: %v", err)
	}

	p.Audio.songCompleteFunc = p.songCompleted
	p.Audio.AddStatusCallback(p.audioCallback)

	p.Queue.AddQueueChangedCallback(p.queueChanged)
	return p, nil
}

// notify song has completed
func (p *Player) songCompleted() {
	p.songComplete <- true
}

// is download pending / ongoing
func (p *Player) isDownloadingSong() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.downloadingSong
}

func (p *Player) loop() {
	// interval to refresh status. This is the interval the status will be updated.
	ticker := time.NewTicker(time.Second)

	for true {
		select {
		case <-p.StopChan():
			// stop application
			p.Audio.StopMedia()
			break
		case <-p.songComplete:
			// stream / song complete, get next song
			logrus.Debug("song complete")
			p.Queue.songComplete()
			if len(p.Queue.GetQueue()) == 0 {
				p.Audio.StopMedia()
			} else {
				if p.nextSong != nil {
					err := p.Audio.playSongFromReader(*p.nextSong)
					if err != nil {
						logrus.Errorf("play track: %v", err)
					}
					p.nextSong = nil
				} else {
					p.downloadSong(0)
				}
			}
		case status := <-p.audioUpdated:
			logrus.Infof("got audio status: %v", status)
		case <-ticker.C:
			// periodically update status, this will push status to p.audioUpdated
			p.Audio.updateStatus()
			if p.status.Song != nil && p.status.State == models.AudioStatePlaying {
				if (p.status.Song.Duration-p.status.SongPast.Seconds()) < 5 &&
					!p.isDownloadingSong() && p.nextSong == nil && len(p.Queue.GetQueue()) >= 2 {
					p.downloadSong(1)
				}
			}
		case metadata := <-p.songDownloaded:
			if p.status.State == models.AudioStateStopped {
				// download complete, send to audio
				err := p.Audio.playSongFromReader(metadata)
				if err != nil {
					logrus.Errorf("play track: %v", err)
				}
				p.nextSong = nil
			} else {
				p.nextSong = &metadata
			}
		}
	}
}

// download and play next song asynchronously
func (p *Player) downloadSong(index int) {
	if p.isDownloadingSong() || p.Queue.empty() {
		return
	}
	song := p.Queue.GetQueue()[index]

	p.lock.Lock()
	p.downloadingSong = true
	p.lock.Unlock()
	ok := false

	reader, format, err := p.api.Stream(song)
	if err != nil {
		if strings.Contains(err.Error(), "A task was canceled") {
			// server task may fail sometimes, retry
			logrus.Warningf("Failed to download song, retrying: %v", err)
			time.Sleep(time.Second)
			reader, format, err = p.api.Stream(song)
			if err == nil {
				ok = true
			} else {
				logrus.Errorf("retry downloading song: %v", err)
			}
		} else {
			logrus.Errorf("download song: %v", err)
		}
	} else {
		ok = true
	}
	if ok {
		// fill metadata
		albumId := song.GetParent()
		album, err := p.api.GetAlbum(albumId)
		artist := &models.Artist{Name: "unknown artist"}
		var imageId string
		var imageUrl string
		if err != nil {
			logrus.Error("Failed to get album by id: ", err.Error())
			album = &models.Album{Name: "unknown album"}
		} else {
			imageId = album.ImageId
			imageUrl = p.api.GetImageUrl(album.Id, models.TypeAlbum)
		}
		a, err := p.api.GetArtist(album.GetParent())
		if err != nil {
			logrus.Errorf("Failed to get artist by id: %v", err)
		} else {
			artist = a
			f := func() {
				metadata := songMetadata{
					song:          song,
					album:         album,
					artist:        artist,
					albumImageUrl: imageUrl,
					albumImageId:  imageId,
					reader:        reader,
					format:        format,
				}
				p.songDownloaded <- metadata
			}
			defer f()
		}
	}

	p.lock.Lock()
	p.downloadingSong = false
	p.lock.Unlock()

	// push song to audio
}

// Next plays next song from queue. Override Audio next to ensure there is track to play and download it
func (p *Player) Next() {
	if len(p.Queue.GetQueue()) > 1 {
		p.StopMedia()
		p.Queue.songComplete()
		go p.downloadSong(0)
	}
}

// Previous plays previous track. Override Audio previous to ensure there is track to play and download it
func (p *Player) Previous() {
	if len(p.Queue.GetHistory(10)) > 0 {
		p.StopMedia()
		p.Queue.playLastSong()
		p.Audio.Previous()
		go p.downloadSong(0)
	}
}

// report audio status to server
func (p *Player) audioCallback(status models.AudioStatus) {
	// Skip reporting if disabled in config
	if config.AppConfig.Player.DisablePlaybackReporting {
		return
	}

	p.lock.RLock()
	lastTime := p.lastApiReport
	p.lock.RUnlock()

	if time.Now().Sub(lastTime) < time.Millisecond*9500 && status.Action == models.AudioActionTimeUpdate {
		// jellyfin server instructs to update every 10 sec
		return
	}

	if status.State == models.AudioStateStopped && status.Action == models.AudioActionTimeUpdate {
		// don't report TimeUpdate if player is stopped
		return
	}

	p.lock.Lock()
	p.lastApiReport = time.Now()
	p.lock.Unlock()

	apiStatus := &interfaces.ApiPlaybackState{
		Event:          "",
		ItemId:         "",
		IsPaused:       false,
		IsMuted:        status.Muted,
		PlaylistLength: 0,
		Position:       status.SongPast.Seconds(),
		Volume:         int(status.Volume),
		Shuffle:        status.Shuffle,
	}

	switch status.Action {
	case models.AudioActionStop:
		apiStatus.Event = interfaces.EventStop // Reverted back to interfaces
	case models.AudioActionPlay:
		apiStatus.Event = interfaces.EventStart
	case models.AudioActionNext:
		apiStatus.Event = interfaces.EventAudioTrackChange
	case models.AudioActionPrevious:
		apiStatus.Event = interfaces.EventAudioTrackChange
	case models.AudioActionSetVolume:
		apiStatus.Event = interfaces.EventVolumeChange
	case models.AudioActionTimeUpdate:
		apiStatus.Event = interfaces.EventTimeUpdate
	case models.AudioActionPlayPause:
		if status.Paused {
			apiStatus.Event = interfaces.EventPause
		} else {
			apiStatus.Event = interfaces.EventUnpause
		}
	case models.AudioActionShuffleChanged:
		apiStatus.Event = interfaces.EventShuffleModeChange
	default:
		apiStatus.Event = interfaces.EventTimeUpdate
		logrus.Warningf("cannot map audio state to browser event: %v", status.Action)
	}

	songs := p.GetQueue()
	queue := make([]models.Id, len(songs))
	for i, v := range songs {
		queue[i] = v.Id
	}
	apiStatus.Queue = queue
	apiStatus.IsPaused = status.Paused

	if status.Song != nil {
		apiStatus.ItemId = status.Song.Id.String()
		apiStatus.PlaylistLength = status.Song.Duration
	}
	f := func() {
		// Type assert p.api to interfaces.Api to call ReportProgress
		if reporter, ok := p.api.(interfaces.Api); ok {
			err := reporter.ReportProgress(apiStatus)
			if err != nil {
				logrus.Errorf("report audio progress to server: %v", err)
			}
		} else {
			logrus.Warnf("MediaServer does not implement ReportProgress")
		}
	}
	go f()
}

func (p *Player) queueChanged(queue []*models.Song) {
	// if player has nothing to play, start download
	state := p.Audio.getStatus()
	if state.State == models.AudioStateStopped && len(queue) > 0 {
		go p.downloadSong(0)
	}
}

func (p *Player) Reorder(index int, left bool) bool {
	// do not allow ongoing song to be reordered
	if p.status.State == models.AudioStatePlaying {
		if index == 0 {
			return false
		}
		if index == 1 && left {
			return false
		}
	}

	return p.Queue.Reorder(index, left)
}

func (p *Player) SetShuffle(enabled bool) {
	p.Queue.SetShuffle(enabled)
	p.Audio.SetShuffle(enabled)
}
