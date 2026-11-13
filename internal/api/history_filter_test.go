package api

import (
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/tests"
)

// notTimestampOnly is the SQL condition that hides updates which changed
// nothing but `updated`. It runs here against the test app's SQLite, on
// literal rows, so it needs no audit_logs collection.
func TestNotTimestampOnly(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	const a = `{"body":"x","id":"r1","title":"T","updated":"2026-01-01 00:00:00.000Z"}`
	const b = `{"body":"x","id":"r1","title":"T","updated":"2026-02-01 00:00:00.000Z"}`
	const c = `{"body":"y","id":"r1","title":"T","updated":"2026-02-01 00:00:00.000Z"}`

	cases := []struct {
		name          string
		event         string
		before, after any
		keep          bool
	}{
		{"timestamp-only update is hidden", "update", a, b, false},
		{"body change is kept", "update", a, c, true},
		{"create is kept", "create", nil, a, true},
		{"create with null before is kept", "create", "null", a, true},
		{"update without snapshots is kept", "update", nil, nil, true},
		{"update with one snapshot is kept", "update", a, nil, true},
		{"update with invalid JSON is kept", "update", "not json", "not json", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var keep bool
			err := app.DB().NewQuery(
				"SELECT " + notTimestampOnly + " FROM (SELECT {:et} AS event_type, {:b} AS before_changes, {:a} AS after_changes)",
			).Bind(dbx.Params{"et": tc.event, "b": tc.before, "a": tc.after}).Row(&keep)
			if err != nil {
				t.Fatal(err)
			}
			if keep != tc.keep {
				t.Errorf("keep = %v, want %v", keep, tc.keep)
			}
		})
	}
}
