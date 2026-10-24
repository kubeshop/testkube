// Package tools holds the MCP tool definitions and the client interfaces they call.
//
// # Insights board tools
//
// The board tools (boards.go) take their rules from the boards package and add how a board
// is read and written.
//
// Boards are organization-scoped and the Control Plane refuses API tokens on every board
// endpoint, so the board tools need a user session. APIClient refuses a tkcapi_ token before
// sending anything and returns ErrBoardsRequireUser.
//
// Every board write reads the board first and resends its description. Current Control Planes
// keep a description an update omits, but older ones clear it, so resending is what keeps it
// on those.
//
// Board updates are optimistic-concurrency writes. Resending a value read earlier (the
// description, a recomputed layout) would overwrite a concurrent edit, so every update -
// update_board and the three report tools - goes through writeBoard:
//
//   - It sends expectedVersion, the board version it read. Every write to a board increments it.
//   - The Control Plane refuses a stale write with 409, and both clients turn that into
//     ErrBoardChanged.
//   - The write is then rebuilt from a fresh read, up to boardWriteAttempts times.
//
// A write's builder must derive everything from the board it is handed, never from an earlier
// read. The token is a counter, not updatedAt: two writes can share a timestamp, and a reused
// token would let a stale write through. A Control Plane that predates versions returns none,
// and the write is then unconditional.
//
// delete_board is deliberately not conditional. It resends nothing it read (the read only
// resolves a slug to the ID it deletes by), and the Control Plane checks visibility and delete
// rights against the board as it is at delete time, so deleting removes the board whatever
// changed since the read, as deleting in the dashboard does.
package tools
