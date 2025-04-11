# Decision Log

*(Record significant architectural or technical decisions made during the project.)*

*   **[YYYY-MM-DD HH:MM:SS] - Decision Summary:**
    *   **Context:** *(Briefly describe the situation or problem)*
    *   **Decision:** *(State the decision made)*
    *   **Rationale:** *(Explain the reasoning behind the decision)*
    *   **Implications:** *(Note any consequences or trade-offs)*

---
*Log:*
*[YYYY-MM-DD HH:MM:SS] - Initial Memory Bank creation.*
*   [2025-04-10 11:21:45] - Decision Summary: Remove UI Browsing Functionality
    *   Context: Need to adapt the player for headless operation, removing UI-specific features.
    *   Decision: Removed the `api.Browser` interface and all its implementations and usages throughout the codebase.
    *   Rationale: The `Browser` interface provided methods for fetching artists, albums, images, searching, etc., which are primarily UI concerns and not needed for a headless player.
    *   Implications: Code related to displaying metadata (images, artist/album details beyond basic song info) and UI-based searching/browsing is removed. The player core logic remains functional for streaming.
*   [2025-04-10 11:49:00] - Decision Summary: Remove Specific Getters from item.go
    *   Context: User feedback questioned the necessity of remaining getters in `api/jellyfin/item.go` after removing the main `Browser` interface.
    *   Decision: Analyzed usages and removed non-essential getters (`GetItem`, `GetFavoriteArtists`, `GetFavoriteAlbums`, `GetGenreAlbums`, etc.) and related helpers, keeping only `GetSongsById`.
    *   Rationale: Further align the codebase with headless operation requirements by removing functions solely related to browsing/UI features (favorites, genres, general item info).
    *   Implications: `api/jellyfin/item.go` is further simplified, focusing only on essential song data retrieval needed for playback functionality.
*   [2025-04-10 12:35:00] - Decision Summary: Remove TUI Remnant Files
    *   Context: Code cleanup after transitioning to headless operation, removing unused TUI-specific components.
    *   Decision: Removed files `config/keybindings.go`, `config/colors.go`, `player/items.go`. Methods in `models/playlist.go` were initially removed but restored due to `models.Item` interface dependency in `api/jellyfin/dtos.go`.
    *   Rationale: These components were identified as unused TUI leftovers via code analysis (dependency checks on `tcell`/`twidgets`, usage searches for exported identifiers and methods).
    *   Implications: Reduced codebase size, removed direct TUI dependencies (`tcell`, `twidgets`). Recommend running `go mod tidy` to clean `go.mod`/`go.sum`.
*   [2025-04-10 13:16:00] - Decision Summary: Remove TUI-specific View, Paging, Sorting, Filtering Code
    *   Context: Refactoring codebase for headless operation, removing unnecessary UI elements identified in `models/common.go`, `api/jellyfin/views.go`, `api/jellyfin/dtos.go`, `api/jellyfin/library.go`, `api/jellyfin/api.go`, `api/jellyfin/params.go`, `config/backends.go`, `config/config.go`, `cmd/env.go`.
    *   Decision: Removed `models.View`, `models.Paging`, `models.Filter`, `models.FilterPlayStatus`, `models.QueryOpts`, `SortMode.Label()`, and related functions/fields (`GetViews`, `GetLatestAlbums`, `GetUserViews`, `musicView`, `setPaging`, `setSorting`, `setFilter`, etc.) across the affected files.
    *   Rationale: These components were directly related to UI presentation (displaying library views, paginating lists, user-facing sorting/filtering options) and are not required for headless functionality.
    *   Implications: Codebase simplified by removing UI-specific logic. API interactions related to these features are removed. Core playback and essential API communication remain.
*   [2025-04-10 16:04:56] - Decision Summary: Improve Stream Performance (Start & Stop)
    *   Context: Debugger analysis indicated potential delays in playback start due to fixed initial buffering and slow stream closing.
    *   Decision(s):
        1. Introduced a new config option `player.initial_buffer_kb` in `config/config.go` to allow user configuration of the initial buffer size, overriding the previous calculation based solely on `http_buffering_s`.
        2. Modified `api.NewStreamDownload` in `api/stream.go` to use this new config value.
        3. Modified `api.NewStreamDownload` to create HTTP requests using `context.WithCancel`.
        4. Modified `api.StreamBuffer.Close` to call the stored `context.CancelFunc` to attempt faster termination of the underlying HTTP request before closing the response body.
    *   Rationale: Making the initial buffer configurable provides flexibility to tune startup performance. Using context cancellation offers a mechanism to potentially interrupt blocking network operations during stream closure, improving responsiveness.
    *   Implications: Users can now adjust `initial_buffer_kb` in their config. Stream closing might be faster, especially in cases of network hangs, though the effectiveness depends on the HTTP client's and server's handling of context cancellation.
