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

package models

import (
	"errors"
)

// Paging struct and methods removed - UI specific.

type SortMode string

const (
	SortAsc  = "ASC"
	SortDesc = "DESC"
)

// Label() method removed - UI specific display text.

// ErrInvalidSort occurs if backend does not support given sorting.
var ErrInvalidSort = errors.New("invalid sort")

// ErrInvalidFilter removed - Filter struct is removed.
// var ErrInvalidFilter = errors.New("invalid filter") // Keep ErrInvalidSort for now

type SortField string

const (
	SortByName       SortField = "Name"
	SortByDate       SortField = "Date"
	SortByArtist     SortField = "Artist"
	SortByAlbum      SortField = "Album"
	SortByPlayCount  SortField = "Most played"
	SortByRandom     SortField = "Random"
	SortByLatest     SortField = "Latest"
	SortByLastPlayed SortField = "Last played"
)

// Sort struct describes sorting (Kept for potential internal API use, but NewSort removed)
type Sort struct {
	Field SortField
	Mode  string // Using string directly instead of SortMode for simplicity if Label is gone
}

// NewSort removed - DefaultQueryOpts removed.

// FilterPlayStatus type and constants removed - UI specific.

// Filter struct and methods removed - UI specific.

// QueryOpts struct and DefaultQueryOpts func removed - UI specific.

// AudioState is audio player state, playing song, stopped
type AudioState int

const (
	// AudioStateStopped, no audio to play
	AudioStateStopped AudioState = iota
	// AudioStatePlaying, playing song
	AudioStatePlaying
)

// AudioAction is an action for audio player, set volume, go to next
type AudioAction int

const (
	// AudioActionTimeUpdate means timed update and no actual action has been taken
	AudioActionTimeUpdate AudioAction = iota
	// AudioActionStop stops playing or paused player
	AudioActionStop
	// AudioActionPlay starts stopped player
	AudioActionPlay
	// AudioActionPlayPause toggles play/pause
	AudioActionPlayPause
	// AudioActionNext plays next song from queue
	AudioActionNext
	// AudioActionPrevious plays previous song from queue
	AudioActionPrevious
	// AudioActionSeek seeks song
	AudioActionSeek
	// AudioActionSetVolume sets volume
	AudioActionSetVolume

	AudioActionShuffleChanged
)

// AudioTick is alias for millisecond
type AudioTick int

func (a AudioTick) Seconds() int {
	return int(a / 1000)
}

func (a AudioTick) MilliSeconds() int {
	return int(a)
}

func (a AudioTick) MicroSeconds() int {
	return int(a) * 1000
}

// AudioVolume is volume level in [0,100]
type AudioVolume int

const (
	AudioVolumeMax = 100
	AudioVolumeMin = 0
)

// InRange returns true if volume is in allowed range
func (a AudioVolume) InRange() bool {
	return a >= AudioVolumeMin && a <= AudioVolumeMax
}

// Add adds value to volume. Negative values are allowed. Always returns volume that's in allowed range.
func (a AudioVolume) Add(vol int) AudioVolume {
	result := a + AudioVolume(vol)
	if result < AudioVolumeMin {
		return AudioVolumeMin
	}
	if result > AudioVolumeMax {
		return AudioVolumeMax
	}
	return result
}

// AudioStatus contains audio player status
type AudioStatus struct {
	State  AudioState
	Action AudioAction

	Song          *Song // Note: Changed from models.Song
	Album         *Album // Note: Changed from models.Album
	Artist        *Artist // Note: Changed from models.Artist
	AlbumImageUrl string

	SongPast AudioTick
	Volume   AudioVolume
	Muted    bool
	Paused   bool
	Shuffle  bool
}

func (a *AudioStatus) Clear() {
	a.Song = nil
	a.Album = nil
	a.Artist = nil
	a.AlbumImageUrl = ""
	a.SongPast = 0
	a.Volume = 0 // Assuming default volume is 0, adjust if needed
}