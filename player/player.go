package player

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"time"
	"tryffel.net/pkg/jellycli/api"
	"tryffel.net/pkg/jellycli/config"
	"tryffel.net/pkg/jellycli/task"
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
	Artist  string
	Album   string
	Song    string
	AudioId string
}

//PlayerState holds data about currently playing song if any
type PlayingState struct {
	State       State
	PlayingType Playtype
	Song        string
	Artist      string
	Album       string

	// Content duration in sec
	CurrentSongDuration int
	CurrentSongPast     int
	PlaylistDuration    int
	PlaylistLeft        int
	// Volume [0,100]
	Volume int
}

const (
	// Player states
	// Stop -> Play -> Pause -> Continue -> Stop
	Play     State = 1
	Continue State = 3
	Pause    State = 2
	Stop     State = 0

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
	chanState          chan PlayingState
	chanStreamComplete chan bool
	ticker             *time.Ticker
	state              PlayingState
	lastAction         *Action
	audio              *audio
}

// NewPlayer constructs new player instance
func NewPlayer(a *api.Api) (*Player, error) {
	p := &Player{
		Api:                a,
		chanAction:         make(chan Action),
		chanState:          make(chan PlayingState),
		chanStreamComplete: make(chan bool),
		ticker:             nil,
		audio:              nil,
	}
	p.audio = newAudio(p.chanStreamComplete)
	err := initAudio()
	if err != nil {
		return p, fmt.Errorf("audio init failed: %v", err)
	}
	p.SetLoop(p.loop)

	reader, err := p.Api.GetSongDirect("id", "mp3")
	if err != nil {
		logrus.Error("failed to request file over http: %v", err)
	} else {
		err = p.audio.newFileStream(reader, FormatMp3)
	}
	p.playMedia()
	p.audio.pause(true)
	p.state.State = Pause
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

func (p *Player) loop() {
	p.ticker = time.NewTicker(updateInterval)
	defer p.ticker.Stop()
	chunkStarted := time.Now()
	for true {
		select {
		case tick := <-p.ticker.C:
			if (tick.Second() - chunkStarted.Second()) >= 10 {
				// Query new chunk every 10 sec
				chunkStarted = time.Now()
			}
			at := p.audio.timePast()
			p.state.CurrentSongPast = int(at.Seconds())

			err := p.audio.streamer.Err()
			if err != nil {
				logrus.Error("error in streamers: ", err.Error())
			}
			p.RefreshState()
		case action := <-p.chanAction:
			logrus.Debug("Player received action")
			// User has requested action
			currentState := p.state.State
			newState := action.State
			if currentState != newState {
				if newState == Pause {
					p.audio.pause(true)
					p.state.State = Pause
				} else if newState == Continue || newState == Play {
					p.audio.pause(false)
					p.state.State = Play
				}
			}
			currentVolume := p.state.Volume
			newVolume := action.Volume
			if currentVolume != newVolume {
				if newVolume > config.AudioMaxVolume {
					newVolume = config.AudioMaxVolume
				} else if newVolume < config.AudioMinVolume {
					newVolume = config.AudioMinVolume
				}

				logrus.Infof("Setting volume to %d %", newVolume)
				p.audio.setVolume(newVolume)
				p.state.Volume = newVolume
			}

			p.lastAction = &action
			p.RefreshState()

		case <-p.chanStreamComplete:
			logrus.Debug("Stream complete")
			p.audio.streamer.Close()
			p.state.State = Stop
			p.RefreshState()
		case <-p.StopChan():
			// Program is stopping
			break
		}
	}
}

//RefreshState pushes current state into state channel
func (p *Player) RefreshState() {
	logrus.Debug("emitting player state")
	p.chanState <- p.state
}

func (p *Player) playMedia() {
	length := p.audio.streamer.Len() / config.AudioSamplingRate
	logrus.Infof("Song length is %d sec.", length)

	err := p.audio.playStream()
	if err != nil {
		logrus.Error("Failed to play media: ", err.Error())
	}
	p.state.State = Play
	p.state.CurrentSongDuration = length
	p.state.CurrentSongPast = 0

}
