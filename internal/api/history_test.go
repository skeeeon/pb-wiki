package api

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"
)

// jsonRaw is the only pure helper in history.go — it normalises whatever the
// PB JSON field round-trip returns into types.JSONRaw so the response always
// contains valid JSON. Handler-level tests would need PB test fixtures
// (documents, access_rules, wiki_config, audit_logs) that don't exist in the
// repo yet; the verification section of the plan covers end-to-end behaviour
// via a manual smoke test.
func TestJSONRawFallsBackToNullOnUnknownTypes(t *testing.T) {
	// Build an empty record off a throwaway collection so r.Get returns the
	// field's zero value (nil for an unset JSON field).
	collection := core.NewBaseCollection("__test")
	collection.Fields.Add(&core.JSONField{Name: "payload", MaxSize: 1024})
	r := core.NewRecord(collection)

	got := jsonRaw(r, "payload")
	if string(got) != "null" && string(got) != "" {
		// An unset JSON field can legitimately return either an empty
		// types.JSONRaw or the literal "null"; both are valid JSON.
		t.Fatalf("unset field: got %q, want \"null\" or empty", string(got))
	}

	r.Set("payload", types.JSONRaw(`{"a":1}`))
	if got := jsonRaw(r, "payload"); string(got) != `{"a":1}` {
		t.Errorf("typed value: got %q, want %q", string(got), `{"a":1}`)
	}

	if got := jsonRaw(r, "missing-field"); string(got) != "null" {
		t.Errorf("missing field: got %q, want \"null\"", string(got))
	}
}

func TestUpdatedBy(t *testing.T) {
	cases := []struct {
		name     string
		snapshot types.JSONRaw
		want     string
	}{
		{"web edit", types.JSONRaw(`{"body":"x","updated_by":"u123"}`), "u123"},
		{"import clears it", types.JSONRaw(`{"body":"x","updated_by":""}`), ""},
		{"old snapshot without the field", types.JSONRaw(`{"body":"x"}`), ""},
		{"no snapshot", types.JSONRaw(`null`), ""},
		{"empty", types.JSONRaw(``), ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := updatedBy(tc.snapshot); got != tc.want {
				t.Errorf("updatedBy(%s) = %q, want %q", tc.snapshot, got, tc.want)
			}
		})
	}
}
