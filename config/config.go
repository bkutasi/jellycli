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

// Package config contains application-wide configurations and constants. Parts of configuration are user-editable
// and per-instance and needs to be persisted. Others are static and meant for tuning the application.
// It also contains some helper methods to read and write config files and create directories when needed.
package config

import (
	"bufio"
	"fmt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh/terminal"
	"os"
	"path"
	"strings"
	"syscall"
	"time"
)

// AppConfig is a configuration loaded during startup
var AppConfig *Config

var configIsEmpty bool

type Config struct {
	Jellyfin Jellyfin `yaml:"jellyfin"`
	Player   Player `yaml:"player"`
	ClientID string `yaml:"client_id"`
}


type Player struct {
	Server                   string `yaml:"server"`
	LogFile                  string `yaml:"log_file"`
	LogLevel                 string `yaml:"log_level"`
	AudioBufferingMs         int    `yaml:"audio_buffering_ms"`
	HttpBufferingS           int    `yaml:"http_buffering_s"`
	// memory limit in MiB
	HttpBufferingLimitMem    int  `yaml:"http_buffering_limit_mem"`
	EnableRemoteControl      bool `yaml:"enable_remote_control"`
	DisablePlaybackReporting bool `yaml:"disable_playback_reporting"`

	LocalCacheDir    string `yaml:"local_cache_dir"`

}


func (p *Player) sanitize() {

	if p.LogFile == "" {
		dir := os.TempDir()
		p.LogFile = path.Join(dir, AppNameLower+".log")
	}
	if p.LogLevel == "" {
		p.LogLevel = logrus.WarnLevel.String()
	}

	if p.AudioBufferingMs == 0 {
		p.AudioBufferingMs = 150
	}
	if p.HttpBufferingS == 0 {
		p.HttpBufferingS = 5
	}
	if p.HttpBufferingLimitMem == 0 {
		p.HttpBufferingLimitMem = 20
	}

	if p.LocalCacheDir == "" {
		baseCacheDir, err := os.UserCacheDir()
		if err != nil {
			logrus.Fatalf("cannot set cache directory, please set manually: 'config.player.local_cache_dir")
		}
		p.LocalCacheDir = path.Join(baseCacheDir, AppNameLower)
	}

}

// initialize new config with some sensible values
func (c *Config) initNewConfig() {
	c.Player.sanitize()
	c.Player.EnableRemoteControl = true
	if c.Player.Server == "" {
		c.Player.Server = "jellyfin"
	}
	c.Player.LogLevel = logrus.InfoLevel.String()

	tempDir := os.TempDir()
	c.Player.LogFile = path.Join(tempDir, "jellycli.log")
}

// can config file be considered empty / not configured
func (c *Config) isEmptyConfig() bool {
	return c.Jellyfin.UserId == "" &&
		c.Player.Server == ""
}

// ReadUserInput reads value from stdin. Name is printed like 'Enter <name>. If mask is true, input is masked.
func ReadUserInput(name string, mask bool) (string, error) {
	fmt.Print("Enter ", name, ": ")
	var val string
	var err error
	if mask {
		// needs cast for windows
		raw, err := terminal.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return "", fmt.Errorf("failed to read user input: %v", err)
		}
		val = string(raw)
		fmt.Println()
	} else {
		reader := bufio.NewReader(os.Stdin)
		val, err = reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read user input: %v", err)
		}
	}
	val = strings.Trim(val, "\n\r")
	return val, nil
}

// ConfigFromViper reads full application configuration from viper.
func ConfigFromViper() error {

	AppConfig = &Config{
		Jellyfin: Jellyfin{
			Url:       viper.GetString("jellyfin.url"),
			Token:     viper.GetString("jellyfin.token"),
			UserId:    viper.GetString("jellyfin.userid"),
			DeviceId:  viper.GetString("jellyfin.device_id"),
			ServerId: viper.GetString("jellyfin.server_id"),
			// MusicView: viper.GetString("jellyfin.music_view"), // Removed: TUI-specific concept
		},
		Player: Player{
			Server:                   viper.GetString("player.server"),
			LogFile:                  viper.GetString("player.logfile"),
			LogLevel:                 viper.GetString("player.loglevel"),
			AudioBufferingMs:         viper.GetInt("player.audio_buffering_ms"),
			HttpBufferingS:           viper.GetInt("player.http_buffering_s"),
			HttpBufferingLimitMem:    viper.GetInt("player.http_buffering_limit_mem"),
			EnableRemoteControl:      viper.GetBool("player.enable_remote_control"),
			DisablePlaybackReporting: viper.GetBool("player.disable_playback_reporting"), // Read new field
			LocalCacheDir:            viper.GetString("player.local_cache_dir"),
		},
		ClientID: viper.GetString("client_id"),
	}

	if AppConfig.Jellyfin.Url == "" {
		configIsEmpty = true
		setDefaults()
	} else {
		AppConfig.Player.sanitize()
	}
	AudioBufferPeriod = time.Millisecond * time.Duration(AppConfig.Player.AudioBufferingMs)
	// VolumeStepSize calculation removed, will be set in settings.go

	// Add debug logging for effective config values
	logrus.Debugf("Effective Config - Player LogLevel: %s", AppConfig.Player.LogLevel)
	logrus.Debugf("Effective Config - Player DisablePlaybackReporting: %t", AppConfig.Player.DisablePlaybackReporting)

	return nil
}

func SaveConfig() error {
	UpdateViper()
	err := viper.WriteConfig()
	if err != nil {
		return fmt.Errorf("save config file: %v", err)
	}
	return nil
}

func setDefaults() {
	if configIsEmpty {
		AppConfig.initNewConfig()
		err := SaveConfig()
		if err != nil {
			logrus.Errorf("save config file: %v", err)
		}
	}
}

// set AppConfig. This is needed for testing.
func configFrom(conf *Config) {
	AppConfig = conf
}

func UpdateViper() {
	viper.Set("jellyfin.url", AppConfig.Jellyfin.Url)
	viper.Set("jellyfin.token", AppConfig.Jellyfin.Token)
	viper.Set("jellyfin.userid", AppConfig.Jellyfin.UserId)
	viper.Set("jellyfin.device_id", AppConfig.Jellyfin.DeviceId)
	viper.Set("jellyfin.server_id", AppConfig.Jellyfin.ServerId)
	// viper.Set("jellyfin.music_view", AppConfig.Jellyfin.MusicView) // Removed: TUI-specific concept

	viper.Set("player.server", AppConfig.Player.Server)
	viper.Set("player.logfile", AppConfig.Player.LogFile)
	viper.Set("player.loglevel", AppConfig.Player.LogLevel)
	viper.Set("player.http_buffering_s", AppConfig.Player.HttpBufferingS)
	viper.Set("player.http_buffering_limit_mem", AppConfig.Player.HttpBufferingLimitMem)
	viper.Set("player.enable_remote_control", AppConfig.Player.EnableRemoteControl)
	viper.Set("player.disable_playback_reporting", AppConfig.Player.DisablePlaybackReporting) // Save new field
	viper.Set("player.audio_buffering_ms", AppConfig.Player.AudioBufferingMs)
	viper.Set("player.local_cache_dir", AppConfig.Player.LocalCacheDir)
	viper.Set("client_id", AppConfig.ClientID)
}

// GetClientID retrieves the unique client ID for this instance.
// If an ID doesn't exist in the config, it generates a new UUID,
// saves it to the config, and returns it.
func GetClientID() (string, error) {
	if AppConfig.ClientID != "" {
		return AppConfig.ClientID, nil
	}

	newID, err := uuid.NewRandom()
	if err != nil {
		return "", fmt.Errorf("failed to generate client UUID: %w", err)
	}

	AppConfig.ClientID = newID.String()
	logrus.Infof("Generated new Client ID: %s", AppConfig.ClientID)

	err = SaveConfig()
	if err != nil {
		// Log the error but proceed, as the ID is generated, just not saved yet.
		// It will be saved on next successful save.
		logrus.Errorf("Failed to save config after generating Client ID: %v", err)
	}

	return AppConfig.ClientID, nil
}
