package api

import (
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/tests"
)

// notTimestampOnly is the SQL condition that hides saves which changed
// nothing but `updated`. It runs here against the test app's SQLite, on
// literal values, so it needs no audit_logs collection.
func TestNotTimestampOnly(t *testing.T) {
	app, err := tests.NewTestApp(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer app.Cleanup()

	cases := []struct {
		name    string
		changed any
		keep    bool
	}{
		{"timestamp-only save is hidden", `["updated"]`, false},
		{"body change is kept", `["body","updated"]`, true},
		{"create is kept", `["body","id","path","title"]`, true},
		{"row from before changed_fields is kept", nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var keep bool
			err := app.DB().NewQuery(
				"SELECT " + notTimestampOnly + " FROM (SELECT {:c} AS changed_fields)",
			).Bind(dbx.Params{"c": tc.changed}).Row(&keep)
			if err != nil {
				t.Fatal(err)
			}
			if keep != tc.keep {
				t.Errorf("keep = %v, want %v", keep, tc.keep)
			}
		})
	}
}
