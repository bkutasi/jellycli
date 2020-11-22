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
	"github.com/denisbrodbeck/machineid"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"tryffel.net/go/jellycli/config"
	"tryffel.net/go/jellycli/interfaces"
	"tryffel.net/go/jellycli/models"
)

const (
	ticksToSecond = int64(10000000)
)

type infoResponse struct {
	ServerName      string `json:"ServerName"`
	Version         string `json:"Version"`
	Id              string `json:"Id"`
	RestartPending  bool   `json:"HasPendingRestart"`
	ShutdownPending bool   `json:"HasShutdownPending"`
}

// GetServerVersion returns name, version, id and possible error
func (a *Api) GetServerVersion() (string, string, string, bool, bool, error) {
	body, err := a.get("/System/Info/Public", nil)
	if err != nil {
		return "", "", "", false, false, fmt.Errorf("request failed: %v", err)
	}

	response := infoResponse{}
	err = json.NewDecoder(body).Decode(&response)
	if err != nil {
		return "", "", "", false, false, fmt.Errorf("response read failed: %v", err)
	}

	return response.ServerName, response.Version, response.Id, response.RestartPending, response.ShutdownPending, nil
}

func (a *Api) VerifyServerId() error {
	_, _, id, _, _, err := a.GetServerVersion()
	if err != nil {
		return err
	}

	if a.serverId != id {
		return fmt.Errorf("server id has changed: expected %s, got %s", a.serverId, id)
	}
	return nil
}

type playbackStarted struct {
	QueueableMediaTypes []string
	CanSeek             bool
	ItemId              string
	MediaSourceId       string
	PositionTicks       int64
	VolumeLevel         int
	IsPaused            bool
	IsMuted             bool
	PlayMethod          string
	PlaySessionId       string
	LiveStreamId        string
	PlaylistLength      int64
	PlaylistIndex       int
	Queue               []queueItem `json:"NowPlayingQueue"`
}

type queueItem struct {
	Id    string `json:"Id"`
	Index string `json:"PlaylistItemId"`
}

func idsToQueue(ids []models.Id) []queueItem {
	out := []queueItem{}
	for i, v := range ids {
		out = append(out, queueItem{
			Id:    v.String(),
			Index: "playlistItem" + strconv.Itoa(i),
		})
	}
	return out
}

type playbackProgress struct {
	playbackStarted
	Event interfaces.ApiPlaybackEvent
}

// ReportProgress reports playback status to server
func (a *Api) ReportProgress(state *interfaces.ApiPlaybackState) error {
	var err error
	var report interface{}
	var url string

	started := playbackStarted{
		QueueableMediaTypes: []string{"Audio"},
		CanSeek:             true,
		ItemId:              state.ItemId,
		MediaSourceId:       state.ItemId,
		PositionTicks:       int64(state.Position) * ticksToSecond,
		VolumeLevel:         state.Volume,
		IsPaused:            state.IsPaused,
		IsMuted:             state.IsPaused,
		PlayMethod:          "DirectPlay",
		PlaySessionId:       a.SessionId,
		LiveStreamId:        "",
		PlaylistLength:      int64(state.PlaylistLength) * ticksToSecond,
		Queue:               idsToQueue(state.Queue),
	}

	if state.Event == interfaces.EventStart {
		url = "/Sessions/Playing"
		report = started
	} else if state.Event == interfaces.EventStop {
		url = "/Sessions/Playing/Stopped"
		report = started
	} else {
		url = "/Sessions/Playing/Progress"
		report = playbackProgress{
			playbackStarted: started,
			Event:           state.Event,
		}
	}

	// webui does not accept websocket response for now, so fall back to http posts. No p
	//if a.socket == nil || state.Event == interfaces.EventStart || state.Event == interfaces.EventStop {
	params := *a.defaultParams()
	body, err := json.Marshal(&report)
	if err != nil {
		return fmt.Errorf("json marshaling failed: %v", err)
	}
	var resp io.ReadCloser
	resp, err = a.post(url, &body, &params)
	if resp != nil {
		resp.Close()
	}

	/*
		} else {
			content := map[string]interface{}{}
			content["MessageType"] = "ReportPlaybackStatus"
			content["Data"] = report

			a.socketLock.Lock()
			a.socket.SetWriteDeadline(time.Now().Add(time.Second * 15))
			err = a.socket.WriteJSON(content)
			a.socketLock.Unlock()
			if err != nil {
				logrus.Errorf("Send playback status via websocket: %v", err)
			}
		}
	*/

	logrus.Debug("Progress event: ", state.Event)

	if err == nil {
		return nil
	} else {
		return fmt.Errorf("push progress: %v", err)
	}
}

func (a *Api) GetCacheItems() int {
	return a.cache.Count()
}

//ImageUrl returns primary image url for item, if there is one. Otherwise return empty
func (a *Api) ImageUrl(item, imageTag string) string {
	return fmt.Sprintf("%s/Items/%s/Images/Primary?maxHeight=500&tag=%s&quality=90", a.host, item, imageTag)
}

func (a *Api) ReportCapabilities() error {
	data := map[string]interface{}{}
	data["PlayableMediaTypes"] = []string{"Audio"}
	data["QueueableMediaTypes"] = []string{"Audio"}
	data["SupportedCommands"] = []string{
		"VolumeUp",
		"VolumeDown",
		"Mute",
		"Unmute",
		"ToggleMute",
		"SetVolume",
	}
	data["SupportsMediaControl"] = a.enableRemoteControl
	data["SupportsPersistentIdentifier"] = false
	data["ApplicationVersion"] = config.Version
	data["Client"] = config.AppName

	data["DeviceName"] = a.deviceName()
	data["DeviceId"] = a.DeviceId

	params := *a.defaultParams()

	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("json: %v", err)
	}

	url := "/Sessions/Capabilities/Full"

	resp, err := a.makeRequest(http.MethodPost, url, &body, &params,
		map[string]string{"X-Emby-Authorization": a.authHeader()})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (a *Api) authHeader() string {
	id, err := machineid.ProtectedID(config.AppName)
	if err != nil {
		logrus.Errorf("get unique host id: %v", err)
		id = randomKey(30)
	}
	hostname := a.deviceName()

	auth := fmt.Sprintf("MediaBrowser Client=\"%s\", Device=\"%s\", DeviceId=\"%s\", Version=\"%s\"",
		config.AppName, hostname, id, config.Version)
	return auth
}

func (a *Api) deviceName() string {
	hostname, err := os.Hostname()
	if err != nil {
		switch runtime.GOOS {
		case "darwin":
			hostname = "mac"
		default:
			hostname = runtime.GOOS
		}
	}
	return hostname
}

func (a *Api) GetLink(item models.Item) string {
	// http://host/jellyfin/web/index.html#!/details.html?id=id&serverId=serverId
	url := fmt.Sprintf("%s/web/index.html#!/details?id=%s", a.host, item.GetId())
	if a.serverId != "" {
		url += "&serverId=" + a.serverId
	}

	return url
}
