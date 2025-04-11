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

package cmd

import (
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "list-env",
	Short: "List env variables",
	Long: `Any configuration variable can be set with environment variables. In addition,
it is also possible to define passwords for servers. This way it would be possible to use
Jellycli without persisting config file (with e.g. Docker). Jellycli will still create config file, nevertheless.

# Config overrides
JELLYCLI_JELLYFIN_URL
JELLYCLI_JELLYFIN_TOKEN
JELLYCLI_JELLYFIN_USERID
JELLYCLI_JELLYFIN_DEVICE_ID
JELLYCLI_JELLYFIN_SERVER_ID
// JELLYCLI_JELLYFIN_MUSIC_VIEW // Removed: TUI-specific concept

JELLYCLI_PLAYER_SERVER
JELLYCLI_PLAYER_LOGFILE
JELLYCLI_PLAYER_LOGLEVEL
JELLYCLI_PLAYER_HTTP_BUFFERING_S
JELLYCLI_PLAYER_HTTP_BUFFERING_LIMIT_MEM
JELLYCLI_PLAYER_AUDIO_BUFFERING_MS
JELLYCLI_PLAYER_ENABLE_REMOTE_CONTROL
JELLYCLI_PLAYER_ENABLE_LOCAL_CACHE
JELLYCLI_PLAYER_ENABLE_LOCAL_CACHE_DIR

# Additional environment variables
JELLYCLI_JELLYFIN_PASSWORD

`,
}

func init() {
	rootCmd.AddCommand(envCmd)

}
