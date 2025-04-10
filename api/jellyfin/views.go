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
	// "tryffel.net/go/jellycli/interfaces" // Removed unused import
	"tryffel.net/go/jellycli/models"
)

func (jf *Jellyfin) GetViews() ([]*models.View, error) {
	params := *jf.defaultParams()

	url := fmt.Sprintf("/Users/%s/Views", jf.userId)
	resp, err := jf.get(url, &params)
	if err != nil {
		return nil, fmt.Errorf("get views: %v", err)
	}
	dto := views{}
	err = json.NewDecoder(resp).Decode(&dto)
	if err != nil {
		return nil, fmt.Errorf("parse views: %v", err)
	}

	views := make([]*models.View, len(dto.Views))
	for i, v := range dto.Views {
		views[i] = v.toView()
	}

	return views, nil
}

func (jf *Jellyfin) GetLatestAlbums() ([]*models.Album, error) {
	params := *jf.defaultParams()
	params["UserId"] = jf.userId
	params.setParentId(jf.musicView)

	resp, err := jf.get(fmt.Sprintf("/Users/%s/Items/Latest", jf.userId), &params)
	if err != nil {
		return nil, fmt.Errorf("request latest albums: %v", err)
	}

	dto := []album{}
	err = json.NewDecoder(resp).Decode(&dto)
	if err != nil {
		return nil, fmt.Errorf("parse latest albums: %v", err)
	}

	albums := make([]*models.Album, len(dto))
	ids := make([]models.Id, len(dto))
	for i, v := range dto {
		albums[i] = v.toAlbum()
		ids[i] = albums[i].Id
	}
	return albums, nil
}


