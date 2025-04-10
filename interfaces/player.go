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

package interfaces

import "tryffel.net/go/jellycli/models"

// Player controls media playback. Current status is sent to StatusCallback, if set. Multiple status callbacks
// can be set.
type Player interface {
	//PlayPause toggles pause
	PlayPause()
	//Pause pauses media that's currently playing. If none, do nothing.
	Pause()
	//Continue continues currently paused media.
	Continue()
	//StopMedia stops playing media.
	StopMedia()
	//Next plays currently next item in queue. If there's no next song available, this method does nothing.
	Next()
	//Previous plays last played song (first in history) if there is one.
	Previous()
	//Seek seeks forward given seconds
	Seek(ticks models.AudioTick)
	//SeekBackwards seeks backwards given seconds
	//AddStatusCallback adds callback that get's called every time status has changed,
	//including playback progress
	AddStatusCallback(func(status models.AudioStatus))
	//SetVolume sets volume to given level in range of [0,100]
	SetVolume(volume models.AudioVolume)
	// SetMute mutes or un-mutes audio
	SetMute(muted bool)
	// ToggleMute toggles current mute.
	ToggleMute()

	SetShuffle(enabled bool)
}

// Queuer contains read-only methods for song queue.
type Queuer interface {
	GetQueue() []*models.Song
	GetTotalDuration() models.AudioTick
}
