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

package player

import (
	"fmt"
	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/flac"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/vorbis"
	"github.com/faiface/beep/wav"
	"github.com/sirupsen/logrus"
	"io"
	"time"
	"tryffel.net/go/jellycli/config"
	"tryffel.net/go/jellycli/interfaces" // Added interfaces import
	"tryffel.net/go/jellycli/models" // Added models import
)

// songMetadata struct moved to player/player.go

// Audio manages playing song and implements interfaces.Player
type Audio struct {
	status models.AudioStatus // Updated to models.AudioStatus

	// todo: we need multiple streamers to allow seamlessly running next song
	streamer beep.StreamSeekCloser

	// ctrl allows pause
	ctrl *beep.Ctrl
	// volume
	volume *effects.Volume
	// mixer allows adding multiple streams sequentially
	mixer *beep.Mixer

	songCompleteFunc func()

	statusCallbacks []func(status models.AudioStatus) // Updated to models.AudioStatus

	currentSampleRate int
}

// initialize new player. This also initializes faiface.Speaker, which should be initialized only once.
func newAudio() *Audio {
	a := &Audio{
		ctrl: &beep.Ctrl{
			Streamer: nil,
			Paused:   false,
		},
		volume: &effects.Volume{
			Streamer: nil,
			Base:     config.AudioVolumeLogBase,
			Volume:   config.AudioMaxVolumedB,
			Silent:   false,
		},
		mixer:           &beep.Mixer{},
		statusCallbacks: make([]func(status models.AudioStatus), 0), // Updated to models.AudioStatus
	}
	a.ctrl.Streamer = a.mixer
	a.ctrl.Paused = false
	a.volume.Streamer = a.ctrl
	a.volume.Silent = false
	a.status.Volume = 100 // Assuming models.AudioVolume is compatible

	a.currentSampleRate = config.AudioSamplingRate
	return a
}

func initAudio() error {
	err := speaker.Init(config.AudioSamplingRate, config.AudioSamplingRate/1000*
		int(config.AudioBufferPeriod.Milliseconds()))
	if err != nil {
		return fmt.Errorf("init speaker: %v", err)
	}
	return nil
}

func (a *Audio) SetShuffle(shuffle bool) {
	if shuffle {
		logrus.Info("Enable shuffle")
	} else {
		logrus.Info("Disable shuffle")
	}

	speaker.Lock()
	defer speaker.Unlock()
	a.status.Shuffle = shuffle
	a.status.Action = models.AudioActionShuffleChanged // Updated to models.AudioAction
	go a.flushStatus()
}

func (a *Audio) getStatus() models.AudioStatus { // Updated return type
	speaker.Lock()
	defer speaker.Unlock()
	return a.status
}

// PlayPause toggles pause.
func (a *Audio) PlayPause() {
	speaker.Lock()
	if a.ctrl == nil {
		speaker.Unlock()
		return
	}
	state := !a.ctrl.Paused
	if state {
		logrus.Info("Pause")
	} else {
		logrus.Info("Continue")
	}
	a.ctrl.Paused = state
	a.status.Paused = state
	a.status.Action = models.AudioActionPlayPause // Updated to models.AudioAction
	speaker.Unlock()
	go a.flushStatus()
}

// Pause pauses audio. If audio is already paused, do nothing.
func (a *Audio) Pause() {
	logrus.Info("Pause audio")
	speaker.Lock()
	if a.ctrl == nil {
		speaker.Unlock()
		return
	}
	a.ctrl.Paused = true
	a.status.Paused = true
	a.status.Action = models.AudioActionPlayPause // Updated to models.AudioAction
	speaker.Unlock()
	go a.flushStatus()
}

// Continue continues paused audio. If audio is already playing, do nothing.
func (a *Audio) Continue() {
	logrus.Info("Continue audio")
	speaker.Lock()
	if a.ctrl == nil {
		speaker.Unlock()
		return
	}
	a.ctrl.Paused = false
	a.status.Paused = false
	a.status.Action = models.AudioActionPlayPause // Updated to models.AudioAction
	speaker.Unlock()
	go a.flushStatus()
}

// StopMedia stops music. If there is no audio to play, do nothing.
func (a *Audio) StopMedia() {
	logrus.Infof("Stop audio")
	speaker.Lock()
	a.status.State = models.AudioStateStopped // Updated to models.AudioState
	a.status.Action = models.AudioActionStop // Updated to models.AudioAction
	a.ctrl.Paused = false
	a.status.Paused = false
	speaker.Unlock()
	speaker.Clear()

	speaker.Lock()
	err := a.closeOldStream()
	speaker.Unlock()
	if err != nil {
		logrus.Errorf("stop: %v", err)
	}
	go a.flushStatus()
}

// Next plays next track. If there's no next song to play, do nothing.
func (a *Audio) Next() {
	logrus.Info("Next song")
	speaker.Lock()
	a.status.Action = models.AudioActionNext // Updated to models.AudioAction
	speaker.Unlock()
	go a.flushStatus()
}

// Previous plays previous track. If previous track does not exist, do nothing.
func (a *Audio) Previous() {
	logrus.Info("Previous song")
	speaker.Lock()
	a.status.Action = models.AudioActionPrevious // Updated to models.AudioAction
	speaker.Unlock()
	go a.flushStatus()
}

// Seek seeks given ticks. If there is no audio, do nothing.
// TODO: Implement Seek functionality using streamer.Seek()
func (a *Audio) Seek(ticks models.AudioTick) { // Updated parameter type
	logrus.Warnf("Seek functionality not yet implemented (seek %d ms)", ticks.MilliSeconds())
	// Example (needs proper calculation and locking):
	// speaker.Lock()
	// if a.streamer != nil {
	// 	 newPos := a.streamer.Position() + a.currentSampleRate.N(ticks * time.Millisecond)
	//   if newPos < a.streamer.Len() && newPos >= 0 {
	//	    a.streamer.Seek(newPos)
	//   }
	// }
	// a.status.Action = models.AudioActionSeek // Updated to models.AudioAction
	// speaker.Unlock()
	// go a.flushStatus()
}

// AddStatusCallback adds a callback that gets called every time audio status is changed, or after certain time.
func (a *Audio) AddStatusCallback(cb func(status models.AudioStatus)) { // Updated parameter type
	a.statusCallbacks = append(a.statusCallbacks, cb)
}

// SetVolume sets volume to given level.
func (a *Audio) SetVolume(volume models.AudioVolume) { // Updated parameter type
	decibels := float64(volumeTodB(int(volume)))
	logrus.Debugf("Set volume to %d %s -> %.2f Db", volume, "%", decibels)
	speaker.Lock()

	// settings volume to 0 does not mute audio, set silent to true
	if decibels <= config.AudioMinVolumedB {
		a.volume.Silent = true
		a.volume.Volume = config.AudioMinVolumedB
		a.status.Volume = models.AudioVolumeMin // Updated const
	} else if decibels >= config.AudioMaxVolumedB {
		a.volume.Volume = config.AudioMaxVolumedB
		a.volume.Silent = false
		a.status.Volume = models.AudioVolumeMax // Updated const
	} else {
		a.volume.Silent = false
		a.volume.Volume = decibels
		a.status.Volume = volume
	}
	a.status.Action = models.AudioActionSetVolume // Updated to models.AudioAction
	speaker.Unlock()
	go a.flushStatus()
}

// SetMute mutes and un-mutes audio
func (a *Audio) SetMute(muted bool) {

	if muted {
		logrus.Info("Mute audio")
	} else {
		logrus.Info("Unmute audio")
	}
	speaker.Lock()
	if a.ctrl == nil {
		speaker.Unlock()
		return
	}
	// Don't pause when muting/unmuting
	// a.ctrl.Paused = false
	a.volume.Silent = muted
	a.status.Muted = muted
	speaker.Unlock()
	go a.flushStatus()
}

func (a *Audio) ToggleMute() {
	logrus.Info("Toggle mute")
	speaker.Lock()
	muted := a.status.Muted
	speaker.Unlock()
	a.SetMute(!muted)
}

func (a *Audio) streamCompleted() {
	logrus.Debug("audio stream complete")
	err := a.closeOldStream()
	if err != nil {
		logrus.Errorf("complete stream: %v", err)
	}
	if a.songCompleteFunc != nil {
		a.songCompleteFunc()
	}
}

func (a *Audio) closeOldStream() error {
	// don't use locking here, since speaker calls streamCompleted, which calls this to close reader
	var err error
	var streamErr error
	if a.streamer != nil {
		streamErr = a.streamer.Err()
		if streamErr != nil {
			if streamErr != io.EOF {
				logrus.Errorf("streamer error: %v", streamErr)
				// Assign streamErr to err only if it's not EOF
				err = streamErr
			} else {
				logrus.Debug("Streamer ended with EOF (expected)")
			}
		}
		closeErr := a.streamer.Close()
		if closeErr != nil {
			if closeErr != io.EOF {
				logrus.Errorf("close streamer error: %v", closeErr)
				// Prioritize close error if streamErr was nil or EOF
				if err == nil || err == io.EOF {
					err = fmt.Errorf("close streamer: %v", closeErr)
				}
			} else {
				logrus.Debug("Streamer closed with EOF")
			}
		} else {
			logrus.Debug("closed old streamer")
		}
		a.streamer = nil
	} else {
		// This might not be an error if StopMedia was called before completion
		logrus.Debug("audio stream completed but streamer is already nil")
	}
	return err
}

// gather latest status and flush it to callbacks
func (a *Audio) updateStatus() {
	past := a.getPastTicks()
	speaker.Lock()
	a.status.SongPast = past
	a.status.Action = models.AudioActionTimeUpdate // Updated to models.AudioAction
	speaker.Unlock()
	a.flushStatus()
}

func (a *Audio) flushStatus() {
	speaker.Lock()
	status := a.status
	speaker.Unlock()
	for _, v := range a.statusCallbacks {
		v(status)
	}
}

// play song from io reader. Only song/album/artist/imageurl are used from status.
func (a *Audio) playSongFromReader(metadata songMetadata) error {
	// decode
	var songFormat beep.Format
	var streamer beep.StreamSeekCloser
	var err error
	switch metadata.format { // Use interfaces.AudioFormat
	case interfaces.AudioFormatMp3: // Use interfaces const
		streamer, songFormat, err = mp3.Decode(metadata.reader)
	case interfaces.AudioFormatFlac: // Use interfaces const
		streamer, songFormat, err = flac.Decode(metadata.reader)
	case interfaces.AudioFormatWav: // Use interfaces const
		streamer, songFormat, err = wav.Decode(metadata.reader)
	case interfaces.AudioFormatOgg: // Use interfaces const
		streamer, songFormat, err = vorbis.Decode(metadata.reader)
	default:
		// Close the reader if format is unknown
		if metadata.reader != nil {
			metadata.reader.Close()
		}
		return fmt.Errorf("unknown audio format: %s", metadata.format)
	}
	if err != nil {
		// Close the reader if decoding failed
		if metadata.reader != nil {
			metadata.reader.Close()
		}
		return fmt.Errorf("decode audio stream: %v", err)
	}

	logrus.Debugf("Song %s samplerate: %d Hz", metadata.song.Name, songFormat.SampleRate.N(time.Second))
	sampleRate := songFormat.SampleRate
	if a.currentSampleRate != sampleRate.N(time.Second) {
		logrus.Debugf("Set samplerate to %d Hz", sampleRate.N(time.Second))
		// Re-initialize speaker with the new sample rate
		// Note: This might cause a small gap or click in audio playback
		speaker.Clear() // Clear buffer before re-init
		err = speaker.Init(sampleRate, sampleRate.N(time.Second)/1000*
			int(config.AudioBufferPeriod.Milliseconds()))
		if err != nil {
			logrus.Errorf("Update sample rate (%d -> %d): %v", a.currentSampleRate, sampleRate.N(time.Second), err)
			// Attempt to continue with old sample rate? Or return error?
			// For now, log error and continue, but audio might be distorted.
			// streamer.Close() // Close the successfully decoded streamer
			// return fmt.Errorf("failed to re-initialize speaker for sample rate %d: %v", sampleRate.N(time.Second), err)
		} else {
			a.currentSampleRate = sampleRate.N(time.Second)
		}
	}
	logrus.Debug("Setting new streamer from ", metadata.format.String())
	if streamer == nil {
		return fmt.Errorf("empty streamer after decode") // Should not happen if err is nil
	}

	// streamer variable holds the original StreamSeekCloser (mp3.Decode, etc.)
	// finalStreamer will hold the stream to be played (potentially resampled)
	var finalStreamer beep.Streamer = streamer // Start with the original streamer

	// Ensure the streamer is resampled to the speaker's current sample rate if they differ
	if songFormat.SampleRate != beep.SampleRate(a.currentSampleRate) {
		logrus.Warnf("Resampling stream from %d Hz to %d Hz", songFormat.SampleRate.N(time.Second), a.currentSampleRate)
		// Assign the *beep.Resampler (which is a beep.Streamer) to finalStreamer
		finalStreamer = beep.Resample(4, songFormat.SampleRate, beep.SampleRate(a.currentSampleRate), streamer)
	}

	// Use finalStreamer (which is always a beep.Streamer) for playback sequence
	stream := beep.Seq(finalStreamer, beep.Callback(a.streamCompleted))
	speaker.Clear()
	speaker.Lock()
	old := a.streamer
	a.mixer.Clear()
	a.streamer = streamer // Store the original streamer for seeking? Or resampled? Let's store original for now.
	a.mixer.Add(stream)
	// Start playback unpaused
	a.ctrl.Paused = false
	a.status.Paused = false
	speaker.Unlock()

	// Close the old stream *after* unlocking to avoid deadlock potential
	if old != nil {
		closeErr := old.Close()
		if closeErr != nil && closeErr != io.EOF {
			logrus.Errorf("failed to close old stream: %v", closeErr)
			// Don't overwrite the main error (if any)
			if err == nil {
				err = fmt.Errorf("failed to close old stream: %v", closeErr)
			}
		}
	}

	speaker.Play(a.volume)
	speaker.Lock()

	a.status.Song = metadata.song
	a.status.Album = metadata.album
	a.status.Artist = metadata.artist
	a.status.AlbumImageUrl = metadata.albumImageUrl
	a.status.State = models.AudioStatePlaying // Updated to models.AudioState
	a.status.Action = models.AudioActionPlay // Updated to models.AudioAction
	speaker.Unlock()
	a.flushStatus()
	return err
}

// linear scaling with a & b coefficients
var volumeTodBA = float32(config.AudioMaxVolumedB-config.AudioMinVolumedB) /
	(config.AudioMaxVolume - config.AudioMinVolume)
var volumeTodBB = float32(config.AudioMinVolumedB - config.AudioMinVolume)

// Transform volume to db
func volumeTodB(volume int) float32 {
	return volumeTodBA*float32(volume) + volumeTodBB
}

// how many ticks current track has played
func (a *Audio) getPastTicks() models.AudioTick { // Updated return type
	speaker.Lock()
	defer speaker.Unlock()
	if a.streamer == nil || a.currentSampleRate == 0 {
		return 0
	}
	// Use currentSampleRate for position calculation
	position := a.streamer.Position()
	if position < 0 { // Position might be -1 if streamer is invalid/closed
		return 0
	}
	duration := time.Duration(position) * time.Second / time.Duration(a.currentSampleRate)
	return models.AudioTick(duration.Milliseconds())
}

// AudioFormat definitions moved to interfaces/audio_format.go
