// Package boards holds the transport-free logic behind the Insights board MCP
// tools: the shape of a report's params, the translation of a report into the
// insight query that renders it, and the board layout.
//
// Report params have no backend schema - the Control Plane stores them as an
// opaque JSON object and only the dashboard interprets them. The rules here are
// a port of the dashboard's report types and filter mapping, so a report the
// MCP writes renders and edits in the UI, and render_board shows the numbers
// the UI shows. Both the CLI client and the Control Plane's hosted MCP use this
// package; neither side should reimplement it.
//
// # Keeping the port in step with the dashboard
//
// The ported files are utils/insights.ts, DynamicFilters/types.ts and
// reports/*/type.ts in testkube-cloud-api's js/packages/web. A dashboard change
// to any of them needs a matching change here, and the Control Plane's
// HandlerClient must call this package rather than reimplement it.
// testdata/translation_cases.json pins the report-to-/insights/* translation.
//
// # Time zones
//
// The dashboard ends a day, week, month or quarter range at the viewer's local
// midnight, so a relative range is anchored in a time zone: render_board takes
// an IANA timeZone (default UTC) and passes it as QueryOptions.Location. The
// package embeds time/tzdata because the MCP also runs from images without a
// zoneinfo database.
package boards
