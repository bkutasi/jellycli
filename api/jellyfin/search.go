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

package jellyfin

import (
	"encoding/json"
	"fmt"
	"io"
	"tryffel.net/go/jellycli/models"
)

type SearchHint struct {
	Id          string `json:"Id"`
	Name        string `json:"Name"`
	Year        int    `json:"ProductionYear"`
	Type        string `json:"Type"`
	Duration    int    `json:"RunTimeTicks"`
	Album       string `json:"Album"`
	AlbumId     string `json:"AlbumId"`
	AlbumArtist string `json:"AlbumArtist"`
}

type SearchResult struct {
	Items []SearchHint `json:"SearchHints"`
}

func searchDtoToItems(rc io.ReadCloser, target mediaItemType) ([]models.Item, error) {

	var result itemMapper

	switch target {
	case mediaTypeSong:
		result = &songs{}
	case mediaTypeAlbum:
		result = &albums{}
	case mediaTypeArtist:
		result = &artists{}
	case mediaTypePlaylist:
		result = &playlists{}
	default:
		return nil, fmt.Errorf("unknown item type: %s", target)
	}

	err := json.NewDecoder(rc).Decode(result)
	if err != nil {
		return nil, fmt.Errorf("decode item %s: %v", target, err)
	}

	return result.Items(), nil
}

