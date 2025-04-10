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

// Package interfaces contains interfaces that multiple packages use and communicate with.
package interfaces

import (
	"tryffel.net/go/jellycli/models"
)

// QueueController controls queue and history. Queue shows only upcoming songs and first item in queue is being
// currently played. When moving to next item in queue, first item is moved to history.
// If no queueChangedCallback is set, no queue updates will be returned
type QueueController interface {
	//GetQueue gets currently ongoing queue of items with complete info for each song
	GetQueue() []*models.Song
	//ClearQueue clears queue. This also calls QueueChangedCallback. If first = true, clear also first item. Else
	// leave it as it is.
	ClearQueue(first bool)
	//AddSongs adds songs to the end of queue.
	//Adding songs calls QueueChangedCallback
	AddSongs([]*models.Song)

	//PlayNext adds songs to 2nd index in order.
	PlayNext([]*models.Song)
	//Reorder sets item in index currentIndex to newIndex.
	//If either currentIndex or NewIndex is not valid, do nothing.
	//On successful order QueueChangedCallback gets called.

	// Reorder shifts item in current index to left or right (earlier / later) by one depending on left.
	// If down, play it earlier, else play it later. Returns true if reorder was made.
	Reorder(currentIndex int, down bool) bool
	//GetHistory get's n past songs that has been played.
	GetHistory(n int) []*models.Song
	//AddQueueChangedCallback sets function that is called every time queue changes.
	AddQueueChangedCallback(func(content []*models.Song))

	// RemoveSongs remove song in given index. First index is 0.
	RemoveSong(index int)

	// SetHistoryChangedCallback sets a function that gets called every time history items update
	SetHistoryChangedCallback(func(songs []*models.Song))
}

//MediaManager manages media: artists, albums, songs
