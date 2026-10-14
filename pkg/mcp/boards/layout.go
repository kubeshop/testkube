package boards

import (
	"encoding/json"
	"fmt"
	"strings"
)

// LayoutVersion is the only board layout version the dashboard understands.
const LayoutVersion = 1

// Layout places a board's reports on rows. Each cell holds one report ID.
type Layout struct {
	Version int   `json:"version"`
	Rows    []Row `json:"rows"`
}

// Row is one row of a board layout.
type Row struct {
	ID    string `json:"id"`
	Cells []Cell `json:"cells"`
}

// Cell is one slot of a row, holding a report.
type Cell struct {
	ID string `json:"id"`
}

// ParseLayout decodes a board layout. A missing or null layout is an empty
// version 1 layout, as the Control Plane creates for a new board.
func ParseLayout(raw json.RawMessage) (Layout, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" || trimmed == "{}" {
		return Layout{Version: LayoutVersion, Rows: []Row{}}, nil
	}
	var l Layout
	if err := json.Unmarshal(raw, &l); err != nil {
		return Layout{}, fmt.Errorf("layout is malformed: %w", err)
	}
	if l.Rows == nil {
		l.Rows = []Row{}
	}
	return l, nil
}

// Without returns the layout with the report's cell removed and any row left
// empty dropped. The Control Plane does not touch the layout when a report is
// deleted, so the caller must send this alongside the delete.
func (l Layout) Without(reportID string) Layout {
	out := Layout{Version: l.Version, Rows: make([]Row, 0, len(l.Rows))}
	for _, row := range l.Rows {
		cells := make([]Cell, 0, len(row.Cells))
		for _, c := range row.Cells {
			if c.ID != reportID {
				cells = append(cells, c)
			}
		}
		if len(cells) > 0 {
			out.Rows = append(out.Rows, Row{ID: row.ID, Cells: cells})
		}
	}
	return out
}

// Validate checks a caller-supplied layout against the board's reports: it
// must be version 1, reference only existing reports, place each at most once,
// and leave none out. Missing row IDs are generated.
func (l *Layout) Validate(reportIDs []string) error {
	if l.Version != LayoutVersion {
		return fmt.Errorf("layout version must be %d, got %d", LayoutVersion, l.Version)
	}
	known := map[string]bool{}
	for _, id := range reportIDs {
		known[id] = true
	}
	seen := map[string]bool{}
	for i := range l.Rows {
		if l.Rows[i].ID == "" {
			l.Rows[i].ID = newID()
		}
		if len(l.Rows[i].Cells) == 0 {
			return fmt.Errorf("layout row %d has no cells; drop empty rows", i)
		}
		for _, c := range l.Rows[i].Cells {
			if !known[c.ID] {
				return fmt.Errorf("layout references unknown report %q", c.ID)
			}
			if seen[c.ID] {
				return fmt.Errorf("layout places report %q more than once", c.ID)
			}
			seen[c.ID] = true
		}
	}
	var missing []string
	for _, id := range reportIDs {
		if !seen[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("layout leaves out reports %s; every report must be placed (use remove_board_report to delete one)", strings.Join(missing, ", "))
	}
	return nil
}

// Order returns the report IDs in the order the dashboard shows them - row by
// row, cell by cell - followed by the reports the layout does not place, which
// the dashboard does not show at all.
func (l Layout) Order(reportIDs []string) (placed, unplaced []string) {
	known := map[string]bool{}
	for _, id := range reportIDs {
		known[id] = true
	}
	seen := map[string]bool{}
	for _, row := range l.Rows {
		for _, c := range row.Cells {
			if known[c.ID] && !seen[c.ID] {
				placed = append(placed, c.ID)
				seen[c.ID] = true
			}
		}
	}
	for _, id := range reportIDs {
		if !seen[id] {
			unplaced = append(unplaced, id)
		}
	}
	return placed, unplaced
}

// RowIDs returns the report IDs of each row, for a compact view of the layout.
func (l Layout) RowIDs() [][]string {
	out := make([][]string, 0, len(l.Rows))
	for _, row := range l.Rows {
		ids := make([]string, 0, len(row.Cells))
		for _, c := range row.Cells {
			ids = append(ids, c.ID)
		}
		out = append(out, ids)
	}
	return out
}
