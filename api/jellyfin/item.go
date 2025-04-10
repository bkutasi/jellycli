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
	"tryffel.net/go/jellycli/models"
)

func (jf *Jellyfin) GetSongsById(ids []models.Id) ([]*models.Song, error) {
	params := *jf.defaultParams()
	params.setIncludeTypes(mediaTypeSong)
	params.enableRecursive()

	if len(ids) == 0 {
		return []*models.Song{}, fmt.Errorf("ids cannot be empty")
	}

	idList := ""
	for i, v := range ids {
		if i > 0 {
			idList += ","
		}
		idList += v.String()
	}

	params["Ids"] = idList

	resp, err := jf.get(fmt.Sprintf("/Users/%s/Items", jf.userId), &params)
	if resp != nil {
		defer resp.Close()
	}

	if err != nil {
		return []*models.Song{}, err
	}

	dto := songs{}
	err = json.NewDecoder(resp).Decode(&dto)
	if err != nil {
		return []*models.Song{}, fmt.Errorf("decode json: %v", err)
	}

	songs := make([]*models.Song, len(dto.Songs))

	for i, v := range dto.Songs {
		logInvalidType(&v, "get songs")
		songs[i] = v.toSong()
		songs[i].Index = i + 1
	}

	return songs, nil
}











