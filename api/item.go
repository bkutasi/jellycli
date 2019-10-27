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

package api

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"tryffel.net/pkg/jellycli/models"
)

const (
	defaultLimit = "100"
)

func itemType(dto *map[string]interface{}) (models.ItemType, error) {
	field := (*dto)["Type"]
	text, ok := field.(string)
	if !ok {
		return "", fmt.Errorf("invalid type: %v", text)
	}
	switch text {
	case mediaTypeArtist:
		return models.TypeArtist, nil
	case mediaTypeAlbum:
		return models.TypeAlbum, nil
	case mediaTypeSong:
		return models.TypeSong, nil
	default:
		return "", fmt.Errorf("unknown type: %s", text)
	}
}

func (a *Api) GetItem(id models.Id) (models.Item, error) {
	item, found := a.cache.Get(id)
	if found && item != nil {
		return item, nil
	}
	params := a.defaultParams()
	(*params)["api_key"] = a.token

	resp, err := a.get(fmt.Sprintf("/Users/%s/Items/%s", a.userId, id), params)
	if err != nil {
		return nil, fmt.Errorf("get item by id: %v", err)
	}
	dto := &map[string]interface{}{}
	err = json.NewDecoder(resp).Decode(dto)
	if err != nil {
		return nil, fmt.Errorf("parse json response: %v", err)
	}

	itemT, err := itemType(dto)
	if err != nil {
		return nil, fmt.Errorf("invalid item type: %v", err)
	}
	//decoder := json.NewDecoder(resp)
	//var item models.Item
	switch itemT {
	case models.TypeAlbum:

	case models.TypeArtist:
	}
	return nil, nil
}

func (a *Api) GetItems(ids []models.Id) ([]models.Item, error) {
	// go through items one by one and check if they're in cache, if not, just get all results from api and update cache
	items := make([]models.Item, len(ids))
	inCache := true
	for i, v := range ids {
		item, found := a.cache.Get(v)
		if item == nil || !found {
			inCache = false
			break
		} else {
			items[i] = item
		}
	}
	if inCache {
		return items, nil
	}

	/*
		Get items from api
	*/
	return nil, nil
}

func (a *Api) GetChildItems(id models.Id) ([]models.Item, error) {
	// get users/<uid>/items/<id>?parentid=<pid>
	return nil, nil
}

func (a *Api) GetParentItem(id models.Id) (models.Item, error) {
	return nil, nil
}

func (a *Api) GetArtist(id models.Id) (models.Artist, error) {
	item, found := a.cache.Get(id)
	// Return cached value if both artist and albums exist
	if found && item != nil {
		artist, ok := item.(*models.Artist)
		if !ok {
			a.cache.Delete(id)
			logrus.Warningf("Found artist %s from cache with invalid type: %s", id, item.GetType())
		} else if artist.Albums != nil {
			if len(artist.Albums) == artist.AlbumCount {
				return *artist, nil
			} else {
				a.cache.Delete(id)
			}
		}
	}

	ar := models.Artist{}

	params := *a.defaultParams()
	params["api_key"] = a.token

	resp, err := a.get(fmt.Sprintf("/Users/%s/Items/%s", a.userId, id), &params)
	if err != nil {
		return ar, fmt.Errorf("get artist: %v", err)
	}
	dto := artist{}
	err = json.NewDecoder(resp).Decode(&dto)
	if err != nil {
		return ar, fmt.Errorf("parse artist: %v", err)
	}

	ar = *dto.toArtist()

	albums, err := a.GetArtistAlbums(id)
	if err != nil {
		return ar, fmt.Errorf("get artist albums: %v", err)
	}

	ids := make([]models.Id, len(albums))
	items := make([]models.Item, len(albums))
	for i, v := range albums {
		ids[i] = v.Id
		items[i] = v
	}

	err = a.cache.PutBatch(items, true)
	if err != nil {
		return ar, fmt.Errorf("store artist albums to cache: %v", err)
	}

	ar.Albums = ids
	a.cache.Put(id, &ar, true)

	return ar, nil
}

//GetArtistAlbums retrieves albums for given artist.
func (a *Api) GetArtistAlbums(id models.Id) ([]*models.Album, error) {
	params := *a.defaultParams()
	params["api_key"] = a.token
	params["IncludeItemTypes"] = "MusicAlbum"
	params["Recursive"] = "true"
	//TODO: use also ContributingAlbumArtistIds
	params["AlbumArtistIds"] = id.String()
	params["Limit"] = defaultLimit
	params["SortBy"] = "ProductionYear"

	resp, err := a.get(fmt.Sprintf("/Users/%s/Items", a.userId), &params)
	if err != nil {
		return nil, fmt.Errorf("get artist albums: %v", err)
	}
	dto := albums{}
	err = json.NewDecoder(resp).Decode(&dto)

	if err != nil {
		return nil, fmt.Errorf("parse response body: %v", err)
	}

	albums := make([]*models.Album, len(dto.Albums))
	for i, v := range dto.Albums {
		albums[i] = v.toAlbum()
	}
	return albums, nil
}

func (a *Api) GetAlbum(id models.Id) (models.Album, error) {
	item, found := a.cache.Get(id)
	// Return cached value if both artist and albums exist
	if found && item != nil {
		album, ok := item.(*models.Album)
		if !ok {
			a.cache.Delete(id)
			logrus.Warningf("Found album %s from cache with invalid type: %s", id, item.GetType())
		} else if album.Songs != nil {
			if len(album.Songs) == album.SongCount {
				return *album, nil
			} else {
				a.cache.Delete(id)
			}
		}
	}

	al := models.Album{}
	params := *a.defaultParams()
	params["api_key"] = a.token

	resp, err := a.get(fmt.Sprintf("/Users/%s/Items/%s", a.userId, id), &params)
	if err != nil {
		return al, fmt.Errorf("get album: %v", err)
	}
	dto := album{}
	err = json.NewDecoder(resp).Decode(&dto)
	if err != nil {
		return al, fmt.Errorf("parse album: %v", err)
	}

	al = *dto.toAlbum()

	songs, err := a.GetAlbumSongs(id)
	if err != nil {
		return al, fmt.Errorf("get albums songs: %v", err)
	}

	ids := make([]models.Id, len(songs))
	items := make([]models.Item, len(songs))
	for i, v := range songs {
		ids[i] = v.Id
		items[i] = v
	}

	err = a.cache.PutBatch(items, true)
	if err != nil {
		return al, fmt.Errorf("store artist albums to cache: %v", err)
	}
	al.SongCount = len(ids)
	al.Songs = ids
	a.cache.Put(id, &al, true)

	return al, nil
}

//GetAlbumSongs gets songs for given album.
func (a *Api) GetAlbumSongs(album models.Id) ([]*models.Song, error) {
	params := *a.defaultParams()
	params["api_key"] = a.token
	params["Recursive"] = "true"
	params["ParentId"] = album.String()
	params["SortBy"] = "IndexNumber"
	params["Limit"] = defaultLimit

	resp, err := a.get(fmt.Sprintf("/Users/%s/Items", a.userId), &params)
	if err != nil {
		return nil, fmt.Errorf("get album Songs; %v", err)
	}

	dto := songs{}
	err = json.NewDecoder(resp).Decode(&dto)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Songs: %v", err)
	}

	songs := make([]*models.Song, len(dto.Songs))
	for i, v := range dto.Songs {
		songs[i] = v.toSong()
	}

	return songs, nil
}

func (a *Api) GetFavoriteArtists() ([]*models.Artist, error) {
	params := *a.defaultParams()
	params["api_key"] = a.token
	params["IsFavorite"] = "true"

	resp, err := a.get("/Artists", &params)
	if err != nil {
		return nil, fmt.Errorf("get favorite artists: %v", err)
	}

	dto := artists{}
	err = json.NewDecoder(resp).Decode(&dto)
	if err != nil {
		return nil, fmt.Errorf("parse artists: %v", err)
	}

	artists := make([]*models.Artist, len(dto.Artists))

	// FavoriteArtists doesn't return any album info
	for i, v := range dto.Artists {
		if v.TotalAlbums == 0 {
			v.TotalAlbums = -1
		}
		artists[i] = v.toArtist()
	}
	return artists, nil
}
