package boards

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Board is a board as the Control Plane returns it from get, create and
// update.
type Board struct {
	ID             string          `json:"id"`
	Slug           string          `json:"slug"`
	Name           string          `json:"name"`
	Description    string          `json:"description,omitempty"`
	CreatedAt      string          `json:"createdAt,omitempty"`
	UpdatedAt      string          `json:"updatedAt,omitempty"`
	Creator        string          `json:"creator,omitempty"`
	Shared         bool            `json:"shared"`
	IsUserFavorite bool            `json:"isUserFavorite,omitempty"`
	IsOrgFavorite  bool            `json:"isOrgFavorite,omitempty"`
	Layout         json.RawMessage `json:"layout,omitempty"`
	Content        struct {
		Reports []Report `json:"reports"`
	} `json:"content"`
}

// ParseBoard decodes a board response.
func ParseBoard(raw string) (*Board, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, fmt.Errorf("empty board response")
	}
	var b Board
	if err := json.Unmarshal([]byte(raw), &b); err != nil {
		return nil, fmt.Errorf("failed to parse board: %w", err)
	}
	return &b, nil
}

// ReportIDs returns the IDs of the board's reports in storage order.
func (b *Board) ReportIDs() []string {
	ids := make([]string, 0, len(b.Content.Reports))
	for _, r := range b.Content.Reports {
		ids = append(ids, r.ID)
	}
	return ids
}

// FindReport returns the board's report with the given ID.
func (b *Board) FindReport(id string) (*Report, bool) {
	for i := range b.Content.Reports {
		if b.Content.Reports[i].ID == id {
			return &b.Content.Reports[i], true
		}
	}
	return nil, false
}

// OrderedReports returns the board's reports in the order the dashboard shows
// them, and the IDs of the reports its layout does not place.
func (b *Board) OrderedReports() (ordered []Report, unplaced []string, err error) {
	layout, err := ParseLayout(b.Layout)
	if err != nil {
		return nil, nil, err
	}
	placed, unplaced := layout.Order(b.ReportIDs())
	for _, ids := range [][]string{placed, unplaced} {
		for _, id := range ids {
			r, _ := b.FindReport(id)
			ordered = append(ordered, *r)
		}
	}
	return ordered, unplaced, nil
}
