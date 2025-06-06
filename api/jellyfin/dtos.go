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
	"fmt"
	"github.com/sirupsen/logrus"
	"tryffel.net/go/jellycli/models"
)

type mediaItemType string

func (m mediaItemType) String() string {
	return string(m)
}

func toItemType(t models.ItemType) mediaItemType {
	switch t {
	case models.TypeArtist:
		return mediaTypeArtist
	case models.TypeAlbum:
		return mediaTypeAlbum
	case models.TypeSong:
		return mediaTypeSong
	case models.TypePlaylist:
		return mediaTypePlaylist
	case models.TypeGenre:
		return mediaTypeGenre
	default:
		return ""
	}
}

const (
	mediaTypeAlbum        mediaItemType = "MusicAlbum"
	mediaTypeArtist       mediaItemType = "MusicArtist"
	mediaTypeSong         mediaItemType = "Audio"
	mediaTypePlaylist     mediaItemType = "Playlist"
	folderTypePlaylists   mediaItemType = "PlaylistsFolder"
	folderTypeCollections mediaItemType = "CollectionFolder"
	mediaTypeGenre        mediaItemType = "Genre"
)

// itemType: each item provided by api has Type-field. This interface returns expected type and actual type
type itemType interface {
	// what type
	ExpectType() mediaItemType
	GotType() mediaItemType
	ModelType() models.Item
}

type itemMapper interface {
	Items() []models.Item
}

// assert type matches expected
func assertType(item itemType) error {
	got := item.GotType()
	expect := item.ExpectType()
	if got == expect {
		return nil
	}

	return fmt.Errorf("expect '%s', got '%s'", expect, got)
}

// make type assertion and log failures.
// action is user for logging.
func logInvalidType(item itemType, action string) {
	err := assertType(item)
	if err != nil {
		logrus.Errorf("type error (%s): %v", action, err)
	}
}

type userData struct {
	PlayCount  int  `json:"PlayCount"`
	IsFavorite bool `json:"IsFavorite"`
	Played     bool `json:"Played"`
}

type nameId struct {
	Name string `json:"Name"`
	Id   string `json:"Id"`
}

type artists struct {
	Artists      []artist `json:"Items"`
	TotalArtists int      `json:"TotalRecordCount"`
}

func (a *artists) Items() []models.Item {
	items := make([]models.Item, len(a.Artists))
	for i, v := range a.Artists {
		items[i] = v.toArtist()
	}
	return items
}

type artist struct {
	Name          string   `json:"Name"`
	Id            string   `json:"Id"`
	TotalDuration int64    `json:"RunTimeTicks"`
	Type          string   `json:"Type"`
	TotalSongs    int      `json:"SongCount"`
	TotalAlbums   int      `json:"AlbumCount"`
	UserData      userData `json:"UserData"`
}

func (a *artist) ExpectType() mediaItemType {
	return mediaTypeArtist
}

func (a *artist) GotType() mediaItemType {
	return mediaItemType(a.Type)
}

func (a *artist) ModelType() models.Item {
	return a.toArtist()
}

func (a *artist) toArtist() *models.Artist {
	return &models.Artist{
		Id:            models.Id(a.Id),
		Name:          a.Name,
		Albums:        nil,
		TotalDuration: int(a.TotalDuration / ticksToSecond),
		AlbumCount:    a.TotalAlbums,
		Favorite:      a.UserData.IsFavorite,
	}
}

type albums struct {
	Albums      []album `json:"Items"`
	TotalAlbums int     `json:"TotalRecordCount"`
}

func (a *albums) Items() []models.Item {
	items := make([]models.Item, len(a.Albums))
	for i, v := range a.Albums {
		items[i] = v.toAlbum()
	}
	return items
}

type album struct {
	Name      string   `json:"Name"`
	Id        string   `json:"Id"`
	Duration  int64    `json:"RunTimeTicks"`
	Year      int      `json:"ProductionYear"`
	Type      string   `json:"Type"`
	Artists   []nameId `json:"AlbumArtists"`
	Overview  string   `json:"Overview"`
	Genres    []string `json:"Genres"`
	ImageTags images   `json:"ImageTags"`
	UserData  userData `json:"UserData"`
}

func (a *album) ExpectType() mediaItemType {
	return mediaTypeAlbum
}

func (a *album) GotType() mediaItemType {
	return mediaItemType(a.Type)
}

func (a *album) ModelType() models.Item {
	return a.toAlbum()
}

func (a *album) toAlbum() *models.Album {
	var artist models.Id
	if len(a.Artists) >= 1 {
		artist = models.Id(a.Artists[0].Id)
	}

	artists := make([]models.IdName, len(a.Artists))
	for i, v := range a.Artists {
		artists[i].Name = v.Name
		artists[i].Id = models.Id(v.Id)
	}

	return &models.Album{
		Id:                models.Id(a.Id),
		Name:              a.Name,
		Year:              a.Year,
		Duration:          int(a.Duration / ticksToSecond),
		Artist:            artist,
		Songs:             nil,
		SongCount:         -1,
		ImageId:           a.ImageTags.Primary,
		DiscCount:         0,
		AdditionalArtists: artists,
		Favorite:          a.UserData.IsFavorite,
	}
}

type songs struct {
	Songs      []song `json:"Items"`
	TotalSongs int    `json:"TotalRecordCount"`
}

func (s *songs) Items() []models.Item {
	items := make([]models.Item, len(s.Songs))
	for i, v := range s.Songs {
		items[i] = v.toSong()
	}
	return items
}

type song struct {
	Name           string   `json:"Name"`
	Id             string   `json:"Id"`
	Duration       int64    `json:"RunTimeTicks"`
	ProductionYear int      `json:"ProductionYear"`
	IndexNumber    int      `json:"IndexNumber"`
	Type           string   `json:"Type"`
	AlbumId        string   `json:"AlbumId"`
	Album          string   `json:"Album"`
	DiscNumber     int      `json:"ParentIndexNumber"`
	Artists        []nameId `json:"ArtistItems"`

	UserData userData `json:"UserData"`
}

func (s *song) ExpectType() mediaItemType {
	return mediaTypeSong
}

func (s *song) GotType() mediaItemType {
	return mediaItemType(s.Type)
}

func (s *song) ModelType() models.Item {
	return s.toSong()
}

func (s *song) toSong() *models.Song {
	artists := make([]models.IdName, len(s.Artists))
	for i, v := range s.Artists {
		artists[i].Name = v.Name
		artists[i].Id = models.Id(v.Id)
	}

	return &models.Song{
		Id:         models.Id(s.Id),
		Name:       s.Name,
		Duration:   int(s.Duration / ticksToSecond),
		Album:      models.Id(s.AlbumId),
		Index:      s.IndexNumber,
		DiscNumber: s.DiscNumber,
		Artists:    artists,
		Favorite:   s.UserData.IsFavorite,
	}
}

type collections struct {
	Collections []collection `json:"Items"`
}

type collection struct {
	Name           string `json:"Name"`
	Id             string `json:"Id"`
	Type           string `json:"Type"`
	CollectionType string `json:"CollectionType"`
}

type playlists struct {
	Playlists []playlist `json:"Items"`
}

func (p *playlists) Items() []models.Item {
	items := make([]models.Item, len(p.Playlists))
	for i, v := range p.Playlists {
		items[i] = v.toPlaylist()
	}
	return items
}

type playlist struct {
	Name     string   `json:"Name"`
	Id       string   `json:"Id"`
	Genres   []string `json:"Genres"`
	Duration int64    `json:"RunTimeTicks"`
	Type     string   `json:"Type"`
	Songs    int      `json:"ChildCount"`
}

func (p *playlist) ExpectType() mediaItemType {
	return mediaTypePlaylist
}

func (p *playlist) GotType() mediaItemType {
	return mediaItemType(p.Type)
}

func (p *playlist) ModelType() models.Item {
	return p.toPlaylist()
}

func (p *playlist) toPlaylist() *models.Playlist {
	return &models.Playlist{
		Id:        models.Id(p.Id),
		Name:      p.Name,
		Duration:  int(p.Duration / ticksToSecond),
		Songs:     nil,
		SongCount: p.Songs,
	}
}

// view, toView, and views structs removed as they were only used by the removed GetViews function.

type images struct {
	Primary string `json:"Primary"`
}
