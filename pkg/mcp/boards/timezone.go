package boards

import (
	"fmt"
	"strings"
	"time"

	// Embed the time zone database: the MCP server also runs from minimal
	// container images that ship without /usr/share/zoneinfo.
	_ "time/tzdata"
)

// ParseTimeZone resolves an IANA time zone name such as "Europe/Berlin".
// An empty name is UTC.
func ParseTimeZone(name string) (*time.Location, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return time.UTC, nil
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("timeZone must be an IANA time zone name such as 'Europe/Berlin' or 'America/New_York': %w", err)
	}
	return loc, nil
}
