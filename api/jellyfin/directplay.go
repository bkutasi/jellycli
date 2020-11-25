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

package jellyfin

import (
	"bytes"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
	"tryffel.net/go/jellycli/config"
	"tryffel.net/go/jellycli/interfaces"
	"tryffel.net/go/jellycli/models"
)

func (a *Api) GetSongUniversal(song *models.Song) (rc io.ReadCloser, format interfaces.AudioFormat, err error) {
	format = interfaces.AudioFormatNil
	params := a.defaultParams()
	ptr := params.ptr()
	ptr["MaxStreamingBitrate"] = "140000000"
	ptr["AudioSamplingRate"] = fmt.Sprint(config.AudioSamplingRate)
	formats := ""
	for i, v := range interfaces.SupportedAudioFormats {
		if i > 0 {
			formats += ","
		}
		formats += v.String()
	}
	ptr["Container"] = formats
	// Every new request requires new playsession
	a.SessionId = randomKey(20)
	ptr["PlaySessionId"] = a.SessionId
	url := a.host + "/Audio/" + song.Id.String() + "/universal"
	var stream *streamBuffer
	stream, err = NewStreamDownload(url, map[string]string{"X-Emby-Token": a.token}, *params, a.client, song.Duration)
	rc = stream
	format, err = mimeToAudioFormat(stream.resp.Header.Get("Content-Type"))
	return
}

func mimeToAudioFormat(mimeType string) (format interfaces.AudioFormat, err error) {
	format = interfaces.AudioFormatNil
	switch mimeType {
	case "audio/mpeg":
		format = interfaces.AudioFormatMp3
	case "audio/flac":
		format = interfaces.AudioFormatFlac
	case "audio/ogg":
		format = interfaces.AudioFormatOgg
	case "audio/wav":
		format = interfaces.AudioFormatWav

	default:
		err = fmt.Errorf("unidentified audio format: %s", mimeType)
	}
	return
}

// streamBuffer is a buffer that reads whole http body in the background and copies it to local buffer.
type streamBuffer struct {
	lock           *sync.Mutex
	url            string
	headers        map[string]string
	params         map[string]string
	client         *http.Client
	buff           *bytes.Buffer
	bitrate        int
	req            *http.Request
	resp           *http.Response
	cancelDownload chan bool
}

func (s *streamBuffer) Read(p []byte) (n int, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	n, err = s.buff.Read(p)
	return
}

func (s *streamBuffer) Close() error {
	logrus.Debug("Close stream download")
	return s.resp.Body.Close()
}

func (s *streamBuffer) Len() int {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.buff.Len()
}

func (s *streamBuffer) SecondsBuffered() int {
	s.lock.Lock()
	defer s.lock.Unlock()
	buffered := s.buff.Len()
	return buffered / s.bitrate
}

func NewStreamDownload(url string, headers map[string]string, params map[string]string,
	client *http.Client, duration int) (*streamBuffer, error) {
	stream := &streamBuffer{
		lock:           &sync.Mutex{},
		url:            url,
		headers:        headers,
		params:         params,
		bitrate:        duration,
		buff:           bytes.NewBuffer(make([]byte, 0, 1024)),
		cancelDownload: make(chan bool),
	}
	if client == nil {
		client = http.DefaultClient
	}
	stream.client = client

	var err error
	stream.req, err = http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return stream, fmt.Errorf("init http request: %v", err)
	}

	for k, v := range headers {
		stream.req.Header.Add(k, v)
	}

	if params != nil {
		q := stream.req.URL.Query()
		for i, v := range params {
			q.Add(i, v)
		}
		stream.req.URL.RawQuery = q.Encode()
	}

	stream.resp, err = stream.client.Do(stream.req)
	if err != nil {
		return stream, fmt.Errorf("make http request: %v", err)

	}
	if stream.resp.StatusCode != 200 {
		return stream, fmt.Errorf("http request error, statuscode: %d", stream.resp.StatusCode)

	}

	sLength := stream.resp.Header.Get("Content-Length")
	length, err := strconv.Atoi(sLength)

	stream.bitrate = length / duration
	for {
		if stream.buff.Len() > stream.bitrate*config.AppConfig.Player.HttpBufferingS {
			break
		}
		failed := stream.readData()
		if failed {
			return stream, fmt.Errorf("initial buffer failed")
		}
	}
	go stream.bufferBackground()
	return stream, err
}

func (s *streamBuffer) bufferBackground() {
	logrus.Debug("Start buffered stream")
	timer := time.NewTimer(time.Millisecond)
	defer timer.Stop()
loop:
	for {
		select {
		case <-timer.C:
			if s.buff.Len()/1024/1024 > config.AppConfig.Player.HttpBufferingLimitMem {
				logrus.Tracef("Buffer is full")
				timer.Reset(time.Second)
			} else {
				if !s.readData() {
					timer.Reset(time.Second)
				} else {
					break loop
				}
			}
		case <-s.cancelDownload:
			logrus.Debug("Stop buffered stream")
			break loop
		}
	}

	close(s.cancelDownload)
	s.cancelDownload = nil
}

func (s *streamBuffer) readData() bool {
	var nHttp int
	var nBuff int
	var err error
	buf := make([]byte, s.bitrate*5)

	s.lock.Lock()
	defer s.lock.Unlock()
	nHttp, err = s.resp.Body.Read(buf)
	stop := false
	if err != nil {
		if err == io.EOF {
			if nHttp == 0 {
				logrus.Debugf("buffer download complete")
				stop = true
			} else {
				// pass
			}
		} else {
			logrus.Errorf("buffer read bytes from body: %v", err)
			stop = true
		}
	}

	buf = buf[0:nHttp]
	if nHttp > 0 {
		nBuff, err = s.buff.Write(buf)
		if err != nil {
			if err == io.EOF {
			} else {
				logrus.Warningf("Copy buffer: %v", err)
			}
		}
		if nBuff != nHttp {
			logrus.Warningf("incomplete buffer read: have %d B, want %d B", nBuff, nHttp)
		}
	}
	size := s.buff.Len()
	logrus.Tracef("Buffer: %d KiB, %d sec", size/1024, size/s.bitrate)
	return stop
}
