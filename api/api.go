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

// Package api contains interface for connecting to remote server. Subpackages contain implementations.
package api

import (
	"io"
	"tryffel.net/go/jellycli/config"
	"tryffel.net/go/jellycli/interfaces"
	"tryffel.net/go/jellycli/models"
)

// MediaServer combines minimal interfaces for browsing and playing songs from remote server.
// Mediaserver can additionally implement RemoteController, and Cacher.
type MediaServer interface {
	Streamer
	RemoteServer
}

// Streamer contains methods for streaming audio from remote location.
type Streamer interface {

	// Stream streams song. If server does not implement separate streaming endpoint,
	// implementcation can wrap Download.
	Stream(Song *models.Song) (io.ReadCloser, interfaces.AudioFormat, error)

	// Download downloads original audio file.
	Download(Song *models.Song) (io.ReadCloser, interfaces.AudioFormat, error)
}


// RemoteController controls audio player remotely as well as
// keeps remote server updated on player status.
type RemoteController interface {
	// SetPlayer allows connecting remote controller to player, which can
	// then be controlled remotely.
	SetPlayer(player interfaces.Player)

	SetQueue(q interfaces.QueueController)

	RemoteControlEnabled() error
}

// RemoteServer contains general methods for getting server connection status
type RemoteServer interface {
	// GetInfo returns general info
	GetInfo() (*models.ServerInfo, error)

	// ConnectionOk returns nil of connection ok, else returns description for failure.
	ConnectionOk() error

	// GetConfig returns backend config that is saved to config file.
	GetConfig() config.Backend

	// Start starts background service for remote server, if any.
	Start() error

	// Stop stops background service for remote server, if any.
	Stop() error

	// GetId returns unique id for server. If server does not provide one,
	// it can be e.g. hashed from url and user.
	GetId() string
}
