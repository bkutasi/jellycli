# Jellycli

[![Godoc Reference](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/tryffel.net/go/jellycli)
[![Go Report Card](https://goreportcard.com/badge/tryffel.net/go/jellycli)](https://goreportcard.com/report/tryffel.net/go/jellycli)

Terminal music player, works with: 
* Jellyfin >= 10.6 (and Emby >= 4.4)
* **Experimental:** Subsonic compatible server, with API >= 1.16 (tested with Navidrome)

![Screenshot](screenshots/browse.png)

## Features

Available features vary depending on server being used. E.g. Subsonic-servers do not support remote control.

* View artists, songs, albums, playlists, favorite artists and albums, genres, similar albums and artists
* Queue: add songs and albums, reorder & delete songs, clear queue
* Control (and view) play state through Dbus integration
* Remote control over Jellyfin server. Currently implemented:
    * [x] Play / pause / stop
    * [x] Set volume
    * [x] Next/previous track
    * [x] Control queue
    * [ ] Seeking, see [#8](https://github.com/tryffel/jellycli/issues/8)
    * [ ] Shuffle, todo
    * [x] Search & filter results
* Supported formats (server transcodes everything else to mp3): mp3,ogg,flac,wav
* headless mode (--no-gui)

**Platforms tested**:
* [x] Windows 10 (amd64)
* [x] Linux 64 bit (amd64)
* [x] Linux 32 bit (armv7 / raspi 2)
* [ ] MacOS

Jellycli (headless & Gui) has been tested and works with Windows. However, there are some limitations, 
namely poor colors, missing characters and some keybindings
might not work as expected. Windows Console works better than Cmd.

On raspi 2 you need to increase audio buffer duration in config file to somewhere around 400.

## Building
**You will need Go 1.13 or later installed and configured**

* For additional audio libraries required, see [Hajimehoshi/oto](https://github.com/hajimehoshi/oto). 
On linux you need libasound2-dev.

Download & build package
```
go get tryffel.net/go/jellycli
cd jellycli
# checkout tag:
# git checkout vx.x.x
go build .
./jellycli
```

## Run
Binaries for 64-bit Linux & Windows are available under
[latest release](https://github.com/tryffel/jellycli/releases/latest).

On Arch Linux you can install Jellycli with pre-built AUR package 'jellycli-bin'.

``` 
# Gui
./jellycli

# Headless mode
./jellycli --no-gui
```

## Docker
Jellycli has experimental docker image tryffel/jellycli. Do note that you might run into issues using audio with docker.
Jellycli relies on alsa and might clash with pulseaudio. In case of problems, 
ensure you have alsa installed on host machine and disable / kill pulseaudio if required. 

```
mkdir ~/jellycli-config
# Gui
docker run -it --rm --device /dev/snd:/dev/snd  -v ~/jellycli-config/jellycli-conf:/root/.config jellycli

# Headless mode
docker run -it --rm --device /dev/snd:/dev/snd  -v ~/jellycli-config/jellycli-conf:/root/.config jellycli --no-gui
```

# Configuration

### Config file

On first time application asks for Jellyfin host, username, password and default collection for music. 

To connect directly to Subsonic, create new config file by running Jellycli for the first time and stop program, 
edit config file and set player.server=subsonic and run Jellycli and insert server info. Alternatively, use env
var JELLYCLI_PLAYER_SERVER=subsonic


All this is stored in configuration file:
* ~/.config/jellycli/jellycli.yaml 
* C:\Users\<user>\AppData\Roaming\jellycli\jellycli.yaml

See config.sample.yaml for more info and up-to-date version of config file.

When Jellycli upgrades existing config file to new version, some values, especially
new boolean have default value 'false', even when the value should be true. 
Be sure to check those values after upgrading application.

Configuration file location is also visible in help page. 
You can use multiple config files by providing argument:
```
jellycli --config temp.yaml
```

Log file is located at '/tmp/jellycli.log' or 'C:\Users\<user>\AppData\Local\Temp/jellycli.log' by default. 
This can be overridden with config file. 
At the moment jellycli does not inform user about errors but rather just silently logs them.
For development purposes you should set log-level either to debug or trace.

### Environment variables:

It is possible to override any config file value with environment variable. In addition to that,
it is also possible to define passwords for servers. This way it would be possible to use
Jellycli without persisting config file (with e.g. Docker). Jellycli will still create config file, nevertheless.

Note: [#14](https://github.com/tryffel/jellycli/issues/14): environment variables override config values, and env variables will be saved in config file.


```

# Config overrides, see config.sample.yaml for more info.
JELLYCLI_JELLYFIN_URL
JELLYCLI_JELLYFIN_TOKEN
JELLYCLI_JELLYFIN_USERID
JELLYCLI_JELLYFIN_DEVICE_ID
JELLYCLI_JELLYFIN_SERVER_ID
JELLYCLI_JELLYFIN_MUSIC_VIEW

JELLYCLI_SUBSONIC_URL
JELLYCLI_SUBSONIC_USERNAME
JELLYCLI_SUBSONIC_SALT
JELLYCLI_SUBSONIC_TOKEN

JELLYCLI_PLAYER_SERVER
JELLYCLI_PLAYER_LOGFILE
JELLYCLI_PLAYER_LOGLEVEL
JELLYCLI_PLAYER_HTTP_BUFFERING_S
JELLYCLI_PLAYER_HTTP_BUFFERING_LIMIT_MEM
JELLYCLI_PLAYER_AUDIO_BUFFERING_MS
JELLYCLI_PLAYER_ENABLE_REMOTE_CONTROL
JELLYCLI_PLAYER_SEARCH_RESULTS_LIMIT

# Additional environment variables
JELLYCLI_GUI_PAGESIZE
JELLYCLI_GUI_DEBUG_MODE
JELLYCLI_GUI_LIMIT_RECENTLY_PLAYED
JELLYCLI_GUI_MOUSE_ENABLED
JELLYCLI_GUI_DOUBLE_CLICK_MS
JELLYCLI_GUI_SEARCH_RESULTS_LIMIT
JELLYCLI_GUI_SEARCH_TYPES
JELLYCLI_GUI_VOLUME_STEPS

JELLYCLI_GUI_ENABLE_SORTING
JELLYCLI_GUI_ENABLE_FILTERING
JELLYCLI_GUI_ENABLE_RESULTS_FILTERING

# Additional environment variables. If Jellycli asks for password (due to failed auth),
# it would normally ask password from user. Supply password here to skip interactive input.
JELLYCLI_JELLYFIN_PASSWORD
JELLYCLI_SUBSONIC_PASSWORD

# disable gui
JELLYCLI_PLAYER_NOGUI
```


### Keybindings & Color Scheme
Keybindings and color scheme are hardcoded at build time. 
They are located in files
* config/keybindings.go
* config/colors.go
edit those as you like. 

To create a debug goroutines dump, enable 'player.debug_mode' 
and then press Ctrl+W to write a text file that's located in log directory. 


## Acknowledgements
Thanks [natsukagami](https://github.com/natsukagami/mpd-mpris) for implementing Mpris-interface.
