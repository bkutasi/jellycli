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
	"math"
	"time"
)

// Paging. First page is 0
type Paging struct {
	TotalItems  int
	TotalPages  int
	CurrentPage int
	PageSize    int
}

// DefaultPaging returns paging with page 0 and default pagesize
// SetTotalItems calculates number of pages for current page size
func (p *Paging) SetTotalItems(count int) {
	p.TotalItems = count
	p.TotalPages = int(math.Ceil(float64(count) / float64(p.PageSize)))
}

// Offset returns offset
func (p *Paging) Offset() int {
	return p.PageSize * p.CurrentPage
}

type SortMode string

const (
	SortAsc  = "ASC"
	SortDesc = "DESC"
)

func (s SortMode) Label() string {
	switch s {
	case SortAsc:
		return "Ascending"
	case SortDesc:
		return "Descending"
	default:
		return "Unknown"
	}
}

// ErrInvalidSort occurs if backend does not support given sorting.
var ErrInvalidSort = errors.New("invalid sort")

// ErrInvalidFilter occurs if backend does not support given filtering.
var ErrInvalidFilter = errors.New("invalid filter")

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

// Sort describes sorting
type Sort struct {
	Field SortField
	Mode  string
}

// NewSort creates default sorting, that is, ASC.
// If field is empty, use SortbyName and ASC
func NewSort(field SortField) Sort {
	if field == "" {
		field = SortByName
	}
	s := Sort{
		Field: field,
		Mode:  SortAsc,
	}
	return s
}

type FilterPlayStatus string

const (
	FilterIsPlayed    = "Played"
	FilterIsNotPlayed = "Not played"
)

// Filter contains filter for reducing results. Some fields are exclusive,
type Filter struct {
	// Played
	FilterPlayed FilterPlayStatus
	// Favorite marks items as being starred / favorite.
	Favorite bool
	// Genres contains list of genres to include.
	Genres []IdName // Note: Changed from models.IdName
	// YearRange contains two elements, items must be within these boundaries.
	YearRange [2]int
}

// YearRangeValid returns true if year range is considered valid and sane.
// If both years are 0, then filter is disabled and range is considered valid.
// Else this checks:
// * 1st year is before or equals 2nd
// * 1st year is after 1900
// * 2nd year if before now() + 10 years
func (f Filter) YearRangeValid() bool {
	if f.YearRange == [2]int{0, 0} {
		return true
	}

	if f.YearRange[0] > f.YearRange[1] {
		return false
	}

	if f.YearRange[0] < 1900 {
		return false
	}

	year := time.Now().Year()
	if f.YearRange[1] > year+10 {
		return false
	}
	return true
}

func (f Filter) Empty() bool {
	return !(f.FilterPlayed == "" && !f.Favorite && len(f.Genres) == 0 && f.YearRange == [2]int{0, 0})
}

type QueryOpts struct {
	Paging Paging
	Filter Filter
	Sort   Sort
}

func DefaultQueryOpts() *QueryOpts {
	return &QueryOpts{
		Paging: Paging{},
		Filter: Filter{},
		Sort: Sort{
			Field: SortByName,
			Mode:  SortAsc,
		},
	}
}

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