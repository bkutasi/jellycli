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

import "fmt"

// AudioFormat represents supported audio formats.
type AudioFormat string

func (a AudioFormat) String() string {
	return string(a)
}

const (
	AudioFormatFlac AudioFormat = "flac"
	AudioFormatMp3  AudioFormat = "mp3"
	AudioFormatOgg  AudioFormat = "ogg"
	AudioFormatWav  AudioFormat = "wav"
	// AudioFormatNil represents an empty format, used for errors or unknown types.
	AudioFormatNil AudioFormat = ""
)

// SupportedAudioFormats lists all audio formats supported by the player backend.
var SupportedAudioFormats = []AudioFormat{
	AudioFormatFlac,
	AudioFormatMp3,
	AudioFormatOgg,
	AudioFormatWav,
}

// MimeToAudioFormat converts a MIME type string to an AudioFormat.
// Returns AudioFormatNil and an error if the MIME type is not recognized.
func MimeToAudioFormat(mimeType string) (format AudioFormat, err error) {
	format = AudioFormatNil
	switch mimeType {
	case "audio/mpeg":
		format = AudioFormatMp3
	case "audio/flac":
		format = AudioFormatFlac
	case "audio/ogg":
		format = AudioFormatOgg
	case "audio/wav":
		format = AudioFormatWav
	default:
		err = fmt.Errorf("unidentified audio format: %s", mimeType)
	}
	return
}