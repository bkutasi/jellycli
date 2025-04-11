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

package api

import (
	"bytes"
	"context" // Import context package
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
	"tryffel.net/go/jellycli/config"
	"tryffel.net/go/jellycli/interfaces" // Changed from player to interfaces
)

// StreamBuffer is a buffer that reads whole http body in the background and copies it to local buffer.
type StreamBuffer struct {
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
	cond           *sync.Cond      // Condition variable for Read
	downloadDone   bool               // Flag indicating download completion/error
	downloadErr    error              // Stores final download error (EOF or other)
	cancelCtx      context.CancelFunc // Function to cancel the underlying HTTP request context
}

func (s *StreamBuffer) Read(p []byte) (n int, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for s.buff.Len() == 0 && !s.downloadDone {
		// Buffer is empty and download is not finished, wait for signal
		logrus.Trace("Read: Buffer empty, waiting for data...")
		s.cond.Wait()
		logrus.Trace("Read: Woke up from wait.")
	}

	// Check buffer again after waking up or if download was already done
	if s.buff.Len() > 0 {
		n, err = s.buff.Read(p)
		// If we read something, return that, even if download finished concurrently.
		// The next Read call will handle the downloadDone state if buffer becomes empty.
		return n, err // err might be io.EOF from buffer, which is fine
	}

	// If buffer is still empty AND download is done, return the download error
	if s.downloadDone {
		logrus.Tracef("Read: Buffer empty, download done. Returning final error: %v", s.downloadErr)
		return 0, s.downloadErr // Return stored error (could be nil or io.EOF)
	}

	// Should theoretically not be reached if logic is correct
	logrus.Error("Read: Reached unexpected state.")
	return 0, io.ErrUnexpectedEOF
}

func (s *StreamBuffer) Close() error {
	logrus.Debug("Close stream download")
	// Signal background buffer to stop if it's running
	// Cancel the request context first
	if s.cancelCtx != nil {
		s.cancelCtx()
		s.cancelCtx = nil // Prevent double cancel
	}

	// Signal background buffer goroutine to stop
	if s.cancelDownload != nil {
		// Use a non-blocking send to avoid deadlock if channel is already closed or receiver isn't ready
		select {
		case s.cancelDownload <- true:
			logrus.Trace("Close: Sent cancel signal to background buffer")
		default:
			logrus.Trace("Close: Cancel signal to background buffer already sent or channel closed")
		}
		close(s.cancelDownload)
		s.cancelDownload = nil // Prevent closing closed channel
	}
	// Close the underlying response body
	if s.resp != nil && s.resp.Body != nil {
		return s.resp.Body.Close()
	}
	return nil // Nothing to close
}


func (s *StreamBuffer) Len() int {
	s.lock.Lock()
	defer s.lock.Unlock()
	// Check if buffer is nil before accessing Len
	if s.buff == nil {
		return 0
	}
	return s.buff.Len()
}

func (s *StreamBuffer) SecondsBuffered() int {
	s.lock.Lock()
	defer s.lock.Unlock()
	// Check for nil buffer and zero bitrate
	if s.buff == nil || s.bitrate == 0 {
		return 0
	}
	buffered := s.buff.Len()
	return buffered / s.bitrate
}

func (s *StreamBuffer) AudioFormat() (format interfaces.AudioFormat, err error) { // Changed player to interfaces
	if s.resp != nil {
		// Call the function now in the interfaces package
		return interfaces.MimeToAudioFormat(s.resp.Header.Get("Content-Type"))
	}
	return interfaces.AudioFormatNil, errors.New("no http response") // Changed player to interfaces
}

func NewStreamDownload(url string, headers map[string]string, params map[string]string,
	client *http.Client, duration int) (*StreamBuffer, error) {
	stream := &StreamBuffer{
		lock:           &sync.Mutex{},
		url:            url,
		headers:        headers,
		params:         params,
		bitrate:        0, // Initialize bitrate, calculate later
		buff:           bytes.NewBuffer(make([]byte, 0, 1024*1024)), // Start with 1MB capacity
		cancelDownload: make(chan bool), // Add missing comma
	}
	stream.cond = sync.NewCond(stream.lock) // Move initialization here
	if client == nil {
		client = http.DefaultClient
	}
	stream.client = client

	// Create a cancellable context for the request
	ctx, cancel := context.WithCancel(context.Background())
	stream.cancelCtx = cancel // Store the cancel function

	var err error
	// Create request with context
	stream.req, err = http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		cancel() // Clean up context if request creation fails
		return nil, fmt.Errorf("init http request with context: %v", err) // Return nil stream on error
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
		return nil, fmt.Errorf("make http request: %v", err) // Return nil stream on error
	}
	if stream.resp.StatusCode != http.StatusOK { // Use http.StatusOK constant
		// Attempt to read body for more details, then close
		bodyBytes, _ := io.ReadAll(stream.resp.Body)
		stream.resp.Body.Close() // Ensure body is closed on error
		return nil, fmt.Errorf("http request error, statuscode: %d, body: %s", stream.resp.StatusCode, string(bodyBytes))
	}

	sLength := stream.resp.Header.Get("Content-Length")
	length, err := strconv.Atoi(sLength)
	if err == nil && duration > 0 && length > 0 {
		stream.bitrate = length / duration // Calculate bitrate in bytes per second
		if stream.bitrate == 0 {
			logrus.Warnf("Calculated bitrate is zero (length: %d, duration: %d)", length, duration)
			// Provide a default reasonable bitrate if calculation fails?
			// stream.bitrate = 128000 / 8 // Example: 128 kbps
		}
	} else {
		logrus.Warnf("Could not calculate bitrate (Content-Length: '%s', duration: %d, parse error: %v)", sLength, duration, err)
		// Provide a default reasonable bitrate if calculation fails?
		// stream.bitrate = 128000 / 8 // Example: 128 kbps
	}

	// Initial buffering logic
	initialBufferTarget := 0
	minBufferBytes := 64 * 1024 // Minimum 64 KiB buffer

	// Prioritize InitialBufferKB if set
	if config.AppConfig.Player.InitialBufferKB > 0 {
		initialBufferTarget = config.AppConfig.Player.InitialBufferKB * 1024 // Convert KiB to Bytes
		logrus.Debugf("Using InitialBufferKB config for initial target: %d bytes", initialBufferTarget)
	} else if stream.bitrate > 0 {
		// Fallback to HttpBufferingS if bitrate is known
		target := stream.bitrate * config.AppConfig.Player.HttpBufferingS
		if target < minBufferBytes {
			initialBufferTarget = minBufferBytes
		} else {
			initialBufferTarget = target
		}
		logrus.Debugf("Using HttpBufferingS config for initial target: %d bytes", initialBufferTarget)
	} else {
		// Fallback to default if bitrate is unknown and InitialBufferKB not set
		initialBufferTarget = 512 * 1024 // Default to 512 KiB
		logrus.Warnf("Bitrate unknown and InitialBufferKB not set. Using default initial buffer target: %d bytes", initialBufferTarget)
	}

	// Ensure the target is at least the minimum
	if initialBufferTarget < minBufferBytes {
		logrus.Warnf("Calculated initial buffer target (%d) is less than minimum (%d). Using minimum.", initialBufferTarget, minBufferBytes)
		initialBufferTarget = minBufferBytes
	}


	for {
		// Check if buffer already meets target before reading
		if stream.buff.Len() >= initialBufferTarget {
			logrus.Debugf("Initial buffer target reached (%d / %d bytes)", stream.buff.Len(), initialBufferTarget)
			break
		}
		finished, readErr := stream.readData() // Update to handle two return values
		if finished {
			// If readData returns true (meaning EOF or error), check buffer size
			if stream.buff.Len() == 0 {
				// Don't return nil stream here, just the error. Close is handled by caller if needed.
				// stream.Close() // Let caller decide if Close is needed based on error
				return nil, fmt.Errorf("initial buffer failed, no data read: %w", readErr) // Wrap original error
			}
			logrus.Warnf("Initial buffering stopped prematurely (%v), buffered %d bytes", readErr, stream.buff.Len()) // Log the error
			break // Stop initial buffering, but proceed if some data was read
		}
	}

	go stream.bufferBackground()
	return stream, nil // Return nil error on success
}

func (s *StreamBuffer) bufferBackground() {
	logrus.Debug("Start background stream buffering")
	// Use a ticker for more regular checks instead of timer resets
	ticker := time.NewTicker(500 * time.Millisecond) // Check every 500ms
	defer ticker.Stop()

loop:
	for {
		select {
		case <-ticker.C:
			// Check buffer limit (use MiB for clarity)
			bufferLimitBytes := config.AppConfig.Player.HttpBufferingLimitMem * 1024 * 1024
			// Check if buffer is nil before accessing Len
			currentLen := 0
			s.lock.Lock()
			if s.buff != nil {
				currentLen = s.buff.Len()
			}
			s.lock.Unlock()

			// Only read if buffer is below the limit
			if currentLen < bufferLimitBytes {
				// REMOVED: s.lock.Unlock() // Unlock before calling readData (which locks internally) - This was incorrect
				logrus.Tracef("Buffer below limit (%d / %d bytes), attempting read", currentLen, bufferLimitBytes)
				readFinished, readErr := s.readData()
				if readFinished {
					s.lock.Lock() // Re-lock to update shared state
					s.downloadDone = true
					s.downloadErr = readErr // Store EOF or actual error
					s.lock.Unlock()         // Unlock after update
					s.cond.Broadcast()      // Wake up any waiting readers
					logrus.Debugf("Background buffering stopped (%v)", readErr)
					break loop
				}
				// Signal readers that new data *might* be available (readData succeeded)
				s.cond.Broadcast()
			} else {
				logrus.Tracef("Buffer limit reached (%d / %d bytes), skipping read this tick", currentLen, bufferLimitBytes)
				// REMOVED: s.lock.Unlock() // Unlock if not reading - This was incorrect
				// Buffer is full, do nothing this tick, wait for reader to consume data
			}
		case <-s.cancelDownload:
			logrus.Debug("Stop background stream buffering requested (cancel signal)")
			s.lock.Lock()
			s.downloadDone = true
			s.downloadErr = io.ErrClosedPipe // Indicate deliberate stop
			s.lock.Unlock()
			s.cond.Broadcast() // Wake up readers
			break loop
		}
	}
	logrus.Debug("Background stream buffering finished loop")
}


// readData reads a chunk from the response body into the buffer.
// Returns true if EOF is reached or an error occurs (signaling the caller to stop).
func (s *StreamBuffer) readData() (finished bool, err error) {
	// This block was duplicated and incorrect, removing it.
	// The correct cancellation check is below.
	// Check if response body exists
	// Check for cancellation signal FIRST
	select {
	case <-s.cancelDownload:
		logrus.Debug("readData: Cancellation signal received before read attempt.")
		return true, io.ErrClosedPipe // Signal stop with specific error
	default:
		// Continue if not cancelled
	}

	if s.resp == nil || s.resp.Body == nil {
		logrus.Error("readData called with nil response body")
		return true, errors.New("response body is nil") // Signal stop with error
	}

	// Determine buffer size dynamically or use a fixed reasonable size
	readChunkSize := 32 * 1024 // Read 32KB chunks
	if s.bitrate > 0 {
		// Read roughly 1 second of data if bitrate is known, capped at e.g., 256KB
		readChunkSize = s.bitrate
		if readChunkSize > 256*1024 {
			readChunkSize = 256 * 1024
		}
		if readChunkSize < 4*1024 { // Ensure a minimum read size
			readChunkSize = 4 * 1024
		}
	}
	buf := make([]byte, readChunkSize)

	nHttp, readErr := s.resp.Body.Read(buf)

	s.lock.Lock() // Lock only when modifying the shared buffer
	// Check if buffer is nil before writing
	if s.buff == nil {
		// Don't unlock yet, need to set downloadDone/Err
		logrus.Error("readData: buffer is nil, cannot write")
		return true, errors.New("buffer is nil") // Return error as well
	}

	if nHttp > 0 {
		nBuff, writeErr := s.buff.Write(buf[:nHttp]) // Write only the bytes read
		if writeErr != nil {
			logrus.Errorf("Error writing to stream buffer: %v", writeErr)
			s.lock.Unlock()
			// Remove commented-out unlock
			return true, writeErr // Treat write error as fatal for buffering
		}
		if nBuff != nHttp {
			logrus.Warnf("Incomplete write to stream buffer: wrote %d B, expected %d B", nBuff, nHttp)
			// Continue buffering, but log the warning
		}
	}
	currentSize := s.buff.Len() // Get size while locked
	s.lock.Unlock()

	// Logging outside the lock
	if nHttp > 0 {
		if currentSize > 0 && s.bitrate > 0 {
			logrus.Tracef("Buffer: %d KiB, ~%d sec, bitrate ~%d kbps", currentSize/1024, currentSize/s.bitrate, s.bitrate*8/1000)
		} else {
			logrus.Tracef("Buffer: %d KiB", currentSize/1024)
		}
	}

	// Remove duplicate unlock, the one at line 348 is correct.

	// Check read error after processing read data and unlocking
	if readErr != nil {
		if readErr == io.EOF {
			logrus.Debug("EOF reached while reading stream body")
		} else {
			logrus.Errorf("Error reading stream body: %v", readErr)
		}
		return true, readErr // Signal stop on EOF or any other read error
	}

	return false, nil // Continue buffering
}