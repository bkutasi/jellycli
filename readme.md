# Jellycli

[![Godoc Reference](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/tryffel.net/go/jellycli)
[![Go Report Card](https://goreportcard.com/badge/tryffel.net/go/jellycli)](https://goreportcard.com/report/tryffel.net/go/jellycli)

Terminal music player for Jellyfin (works with Emby too).

![Screenshot](screenshot.png)

## Features
* View artists, songs, albums, playlists, favorite artists and albums, genres, similar albums and artists
* Queue: add songs and albums, reorder & delete songs, clear queue
* Control (and view) play state through Dbus integration
* Remote control over Jellyfin server. Currently implemented:
    * [x] Play / pause / stop
    * [x] Set volume
    * [x] Next/previous track
    * [x] Control queue
    * [ ] Seeking, see [#8](https://github.com/tryffel/jellycli/issues/8)
* Supported formats (server transcodes everything else to mp3): mp3,ogg,flac,wav
* headless mode (--no-gui)

**Platforms tested**:
* [x] Windows 10 (amd64)
* [x] Linux 64 bit (amd64)
* [x] Linux 32 bit (armv7 / raspi 2)
* [ ] MacOS

Jellycli (headless & Gui) should work on Windows. However, there are some limitations, 
namely poor colors and some keybindings
might not work as expected. Windows Console works better than Cmd.

On raspi 2 you need to increase audio buffer duration in config file to somewhere around 400.

## Building
**You will need Go 1.13 or Go 1.14 installed and configured**

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
Binaries for Linux & Windows are available at 
[latest release](https://github.com/tryffel/jellycli/releases/latest).

On Arch Linux you can install jellycli with pre-built AUR package 'jellycli-bin'.

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

On first time application asks for Jellyfin host, username, password and default collection for music. 
All this is stored in configuration file:
* ~/.config/jellycli/jellycli.yaml 
* C:\Users\<user>\AppData\Roaming\jellycli\jellycli.yaml

See config.sample.yaml for more info and up-to-date version of config file.

Configuration file location is also visible in help page. 
You can use multiple config files by providing argument:
```
jellycli --config temp.yaml
```

Log file is located at '/tmp/jellycli.log' or 'C:\Users\<user>\AppData\Local\Temp/jellycli.log' by default. 
This can be overridden with config file. 
At the moment jellycli does not inform user about errors but rather just silently logs them.
For development purposes you should set log-level either to debug or trace.

Keybindings are hardcoded at build time. They are located in file config/keybindings.go:73 in function 
```
func DefaultKeybindings()
```
edit that function as you like. 

To create debug goroutine dumps, enable 'player.debug_mode' 
and then press Ctrl+W to write a text file that's located in log directory. 


## Acknowledgements
Thanks [natsukagami](https://github.com/natsukagami/mpd-mpris) for implementing Mpris-interface.
