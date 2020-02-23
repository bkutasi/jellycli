package player

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"time"
	"tryffel.net/go/jellycli/api"
	"tryffel.net/go/jellycli/config"
	"tryffel.net/go/jellycli/models"
	"tryffel.net/go/jellycli/task"
)

type State int
type Playtype int
type Status int

//Action includes instruction for player to play
type Action struct {
	// What to do
	State  State
	Type   Playtype
	Volume int

	// Provide either artist/album/song or audio id
	Artist   string
	Album    string
	Song     string
	Year     int
	AudioId  string
	Duration int
}

//PlayerState holds data about currently playing song if any
type PlayingState struct {
	State       State
	PlayingType Playtype
	Song        string
	Artist      string
	Album       string
	Year        int

	// Content duration in sec
	CurrentSongDuration int
	CurrentSongPast     int
	PlaylistDuration    int
	PlaylistLeft        int
	// Volume [0,100]
	Volume int

	CurrentSong *models.Song
}

type PlaySong struct {
	Action Action
	Song   io.ReadCloser
}

const (
	// Player states
	// StopMedia -> Play -> Pause -> (Continue) -> StopMedia
	// Play new song
	Play State = iota
	// Continue paused song, only a transition mode, never state of the player
	Continue
	//SetVolume, only transition mode
	SetVolume
	// Pause song
	Pause
	// StopMedia playing
	Stop

	//SongComplete, only transition mode to notify song has changed
	SongComplete

	// Playing single song
	Song Playtype = 0
	// Playing album
	Album Playtype = 1
	// Playing artists discography
	Artist Playtype = 2
	// Playing playlist
	Playlist Playtype = 3

	// Last action was ok
	StatusOk Status = 0
	// Last action resulted in error
	StatusError Status = 0

	// How often to update state i.e. push status to playingstate channel
	updateInterval = time.Second
)

// Player is the application structure
type Player struct {
	task.Task

	Api *api.Api
	// chanAction is for user interactions
	chanAction chan Action
	// chanState is updated when state is changed
	chanState chan PlayingState

	chanStreamComplete chan bool

	chanAddSong chan PlaySong

	ticker *time.Ticker

	state      PlayingState
	lastAction *Action

	audio  *audio
	reader io.ReadCloser
	itemId string
}

// NewPlayer constructs new player instance
func NewPlayer(a *api.Api) (*Player, error) {
	p := &Player{
		Api:                a,
		chanAction:         make(chan Action, 3),
		chanState:          make(chan PlayingState, 3),
		chanStreamComplete: make(chan bool),
		chanAddSong:        make(chan PlaySong),
		ticker:             nil,
		audio:              nil,
	}
	p.audio = newAudio(p.chanStreamComplete)
	err := initAudio()
	if err != nil {
		return p, fmt.Errorf("audio init failed: %v", err)
	}
	p.SetLoop(p.loop)
	p.Name = "AudioPlayer"

	p.audio.pause(true)
	p.state.State = Stop
	p.state.Volume = 50
	return p, nil
}

//ActionChannel returns input channel for user actions
func (p *Player) ActionChannel() chan Action {
	return p.chanAction
}

//StateChannel return output channel for player state
func (p *Player) StateChannel() chan PlayingState {
	return p.chanState
}

func (p *Player) AddSongChannel() *chan PlaySong {
	return &p.chanAddSong
}

func (p *Player) loop() {
	p.ticker = time.NewTicker(updateInterval)
	defer p.ticker.Stop()
	chunkStarted := time.Now()
	for true {
		select {
		case <-p.ticker.C:
			diff := time.Since(chunkStarted).Seconds()
			if diff >= 10 {
				if p.state.State == Play || p.state.State == Pause {
					go p.reportStatus(api.EventTimeUpdate)
				}
				// Query new chunk every 10 sec
				chunkStarted = time.Now()
			}
			at := p.audio.timePast()
			p.state.CurrentSongPast = int(at.Seconds())
			p.RefreshState()
		case action := <-p.chanAction:
			// User has requested action
			if ok, event := p.handleAction(action); ok {
				p.lastAction = &action
				p.RefreshState()
				go p.reportStatus(event)
			} else {
				logrus.Error("Invalid action, probably incorrect transition")
			}
		case <-p.chanStreamComplete:
			logrus.Debug("Stream complete")
			if p.reader != nil {
				err := p.reader.Close()
				if err != nil {
					logrus.Errorf("Failed to close reader: %v", err)
				}
				p.reader = nil
			}
			p.stop()
			p.state.State = SongComplete
			p.RefreshState()
			p.state.State = Stop
		case <-p.StopChan():
			// Program is stopping
			p.stop()
			break
		case song := <-p.chanAddSong:
			p.playSongFromReader(song)

		}
	}
}

//handle any incoming actions. Return true if state has changed
func (p *Player) handleAction(action Action) (bool, api.PlaybackEvent) {
	defaultEvent := api.EventTimeUpdate
	switch action.State {
	case SetVolume:
		if p.state.Volume != action.Volume && action.Volume != -1 {
			if action.Volume > config.AudioMaxVolume {
				action.Volume = config.AudioMaxVolume
			} else if action.Volume < config.AudioMinVolume {
				action.Volume = config.AudioMinVolume
			}
			p.audio.setVolume(action.Volume)
			p.state.Volume = action.Volume
			go p.reportStatus(api.EventVolumeChange)
			return true, api.EventVolumeChange
		}
	case Pause:
		if p.state.State == Play && p.audio.hasStreamer() {
			p.audio.pause(true)
			p.state.State = Pause
			return true, api.EventPause
		}
		return false, defaultEvent
	case Play:
		if p.state.State == Stop || p.state.State == Pause || p.state.State == Play {
			if p.PlaySong(action) {
				return true, api.EventPlaylistItemAdd
			}
			return false, defaultEvent
		}
	case Stop:
		if p.state.State == Pause || p.state.State == Play {
			p.stop()
			p.state.State = Stop
			return true, api.EventStop
		}
	case Continue:
		if p.state.State == Pause && p.audio.hasStreamer() {
			p.audio.pause(false)
			p.state.State = Play
			return true, api.EventUnpause
		}
		return false, defaultEvent
	default:
		logrus.Error("Got invalid action: ", action.State)
		return false, defaultEvent
	}
	return false, defaultEvent

}

func (p *Player) stop() {
	if p.state.State == Play || p.state.State == Pause {
		p.audio.stop()
		p.reportStatus(api.EventStop)
	}
	p.state.State = Stop
}

//RefreshState pushes current state into state channel
func (p *Player) RefreshState() {
	p.chanState <- p.state
}

func (p *Player) PlaySong(action Action) bool {
	reader, err := p.Api.GetSongDirect(action.AudioId, "mp3")
	if err != nil {
		logrus.Error("failed to request file over http: ", err.Error())
		return false
	} else {
		err = p.audio.newFileStream(reader, FormatMp3)
		if err != nil {
			logrus.Error("Failed to create new stream: ", err.Error())
			return false
		}
	}
	if p.reader != nil {
		p.reader.Close()
	}
	p.itemId = action.AudioId
	p.reader = reader
	p.state.State = Play
	p.state.Song = action.Song
	p.state.Artist = action.Artist
	p.state.Album = action.Album
	p.state.Year = action.Year
	p.state.CurrentSongDuration = action.Duration
	return true
}

func (p *Player) playSongFromReader(play PlaySong) {
	err := p.audio.newFileStream(play.Song, FormatMp3)
	if err != nil {
		logrus.Error("Failed to create new stream: ", err.Error())
		return
	}
	if p.reader != nil {
		p.reader.Close()
	}
	action := play.Action

	p.itemId = action.AudioId
	p.reader = play.Song
	p.state.State = Play
	p.state.Song = action.Song
	p.state.Artist = action.Artist
	p.state.Album = action.Album
	p.state.Year = action.Year
	p.state.CurrentSongDuration = action.Duration
}

func (p *Player) reportStatus(event api.PlaybackEvent) {
	state := &api.PlaybackState{
		Event:          event,
		ItemId:         p.itemId,
		IsPaused:       false,
		IsMuted:        false,
		PlaylistLength: p.state.CurrentSongDuration,
		Position:       p.state.CurrentSongPast,
		Volume:         p.state.Volume,
	}

	if p.state.State == Pause {
		state.IsPaused = true
	}

	err := p.Api.ReportProgress(state)
	if err != nil {
		logrus.Error("Failed to report status: ", err.Error())
	}
}
