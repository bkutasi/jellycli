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

package models

type Id string

func (i Id) String() string {
	return string(i)
}

// Item is any object that has unique id and falls to some category with ItemType.
type Item interface {
	GetId() Id
	GetName() string
	HasChildren() bool
	GetChildren() []Id
	GetParent() Id
	GetType() ItemType
}

type ItemType string

const (
	TypeArtist   ItemType = "Artist"
	TypeAlbum    ItemType = "Album"
	TypePlaylist ItemType = "TypePlaylist"
	TypeQueue    ItemType = "Queue"
	TypeHistory  ItemType = "History"
	TypeSong     ItemType = "Song"
	TypeGenre    ItemType = "Genre"
)
