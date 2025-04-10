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
