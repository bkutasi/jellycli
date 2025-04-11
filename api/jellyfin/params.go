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

package jellyfin

import (
	"strconv"
	// "tryffel.net/go/jellycli/interfaces" // Removed unused import
)

type params map[string]string

// get pointer to map for convinience
func (p *params) ptr() map[string]string {
	return *p
}

// setPaging removed - depends on removed models.Paging

func (p *params) setLimit(n int) {
	(*p)["Limit"] = strconv.Itoa(n)
}

func (p *params) setIncludeTypes(itemType mediaItemType) {
	ptr := p.ptr()
	ptr["IncludeItemTypes"] = itemType.String()
}

func (p *params) enableRecursive() {
	(*p)["Recursive"] = "true"
}

func (p *params) setParentId(id string) {
	(*p)["ParentId"] = id
}

// setSorting removed - depends on removed models.SortMode constants/labels
// setSortingByType removed - depends on removed models.SortMode constants/labels and models.Sort
// setFilter removed - depends on removed models.Filter and models.FilterPlayStatus
// appendFilter removed - only used by setFilter
