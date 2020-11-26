/*
 * Copyright 2020 Tero Vierimaa
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package subsonic

import (
	"fmt"
	"tryffel.net/go/jellycli/models"
)

type subErrCode int

func (err subErrCode) String() string {
	switch err {
	case ErrGeneric:
		return "generic error"
	case ErrParamMissing:
		return "missing parameter"
	case ErrClientProto:
		return "client protocol incompatible"
	case ErrServerProto:
		return "server protocol incompatible"
	case ErrAuth:
		return "invalid authentication"
	case ErrLdap:
		return "ldap error"
	case ErrUnauthorized:
		return "unauthorized"
	case ErrTrialEnded:
		return "server trial ended"
	case ErrNotFound:
		return "record not found"
	}
	return fmt.Sprintf("unkown error code: %d", err)
}

const (
	ErrGeneric      subErrCode = 0
	ErrParamMissing subErrCode = 10
	ErrClientProto  subErrCode = 20
	ErrServerProto  subErrCode = 30
	ErrAuth         subErrCode = 40
	ErrLdap         subErrCode = 41
	ErrUnauthorized subErrCode = 50
	ErrTrialEnded   subErrCode = 60
	ErrNotFound     subErrCode = 70
)

type subError struct {
	Code    subErrCode `json:"code"`
	Message string     `json:"message"`
}

type subResponse struct {
	Resp *response `json:"subsonic-response"`
}

type response struct {
	Status        string        `json:"status"`
	Version       string        `json:"version"`
	Type          string        `json:"type"`
	ServerVersion string        `json:"serverVersion"`
	Error         *subError     `json:"error"`
	MusicFolders  *musicFolders `json:"musicFolders,omitempty"`
	Indexes       *indexes      `json:"indexes,omitempty"`
	Artists       *indexes      `json:"artists,omitempty"`
	Artist        *artistAlbums `json:"artist,omitempty"`
	AlbumList     *albumList    `json:"albumList2,omitempty"`
	Albums        *albumSongs   `json:"album,omitempty"`
	Favorites     *favorites    `json:"starred2,omitempty"`
	Search        *searchResp   `json:"searchResult3,omitempty"`
}

type musicFolder struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type musicFolders struct {
	Folders []musicFolder `json:"musicFolder"`
}

type indexes struct {
	LastModified    int      `json:"lastModified"`
	IgnoredArticles string   `json:"ignoredArticles"`
	Indexes         *[]index `json:"index,omitempty"`
}

type index struct {
	Name    string    `json:"name"`
	Artists *[]artist `json:"artist,omitempty"`
}

type artist struct {
	Id             string  `json:"id"`
	Name           string  `json:"name"`
	AlbumCount     int     `json:"albumCount"`
	Starred        *string `json:"starred"`
	ArtistImageUrl string  `json:"artistImageUrl"`
}

func (a *artist) toArtist() *models.Artist {
	return &models.Artist{
		Id:            models.Id(a.Id),
		Name:          a.Name,
		Albums:        nil,
		TotalDuration: 0,
		AlbumCount:    a.AlbumCount,
		Favorite:      a.Starred != nil,
	}
}

type album struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Artist    string `json:"artist"`
	ArtistId  string `json:"artistId"`
	SongCount int    `json:"songCount"`
	Year      int    `json:"year"`
	Duration  int    `json:"duration"`
	Starred   string `json:"starred"`
}

func (a *album) toAlbum() *models.Album {
	return &models.Album{
		Id:                models.Id(a.Id),
		Name:              a.Name,
		Year:              a.Year,
		Duration:          a.Duration,
		Artist:            models.Id(a.ArtistId),
		AdditionalArtists: nil,
		Songs:             nil,
		SongCount:         a.SongCount,
		ImageId:           "",
		DiscCount:         1,
		Favorite:          a.Starred != "",
	}
}

type artistAlbums struct {
	artist
	Albums []child `json:"album,omitempty"`
}

type albumSongs struct {
	album
	Songs []child `json:"song,omitempty"`
}

type albumList struct {
	Albums []album `json:"album"`
}

type child struct {
	Id         string `json:"id"`
	Parent     string `json:"parent"`
	Title      string `json:"title"`
	Name       string `json:"name"`
	Album      string `json:"album"`
	AlbumId    string `json:"albumId"`
	Artist     string `json:"artist"`
	Track      int    `json:"track"`
	Year       int    `json:"year"`
	Duration   int    `json:"duration"`
	DiscNumber int    `json:"discNumber"`
	ArtistId   string `json:"artistId"`
	Type       string `json:"type"`
	SongCount  int    `json:"songCount"`
}

func (c *child) toAlbum() *models.Album {
	return &models.Album{
		Id:                models.Id(c.Id),
		Name:              c.Title,
		Year:              c.Year,
		Duration:          c.Duration,
		Artist:            models.Id(c.ArtistId),
		AdditionalArtists: nil,
		Songs:             nil,
		SongCount:         c.SongCount,
		ImageId:           "",
		DiscCount:         1,
	}
}

func (c *child) toSong() *models.Song {
	return &models.Song{
		Id:          models.Id(c.Id),
		Name:        c.Title,
		Duration:    c.Duration,
		Index:       c.Track,
		Album:       models.Id(c.AlbumId),
		DiscNumber:  c.DiscNumber,
		Artists:     nil,
		AlbumArtist: models.Id(c.ArtistId),
		Favorite:    false,
	}
}

type searchResp struct {
	Artists []artist `json:"artist,omitempty"`
	Albums  []album  `json:"album,omitempty"`
	Songs   []child  `json:"song"`
}

type favorites struct {
	Artists []artist `json:"artist,omitempty"`
	Albums  []child  `json:"album,omitempty"`
}
