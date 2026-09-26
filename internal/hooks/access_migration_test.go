package hooks_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
)

func migration090(t *testing.T) *core.Migration {
	t.Helper()
	for _, m := range core.AppMigrations.Items() {
		if strings.HasPrefix(m.File, "1700000090_") {
			return m
		}
	}
	t.Fatal("migration 1700000090 not registered")
	return nil
}

// Rolls migration 090 back to path rules, seeds data in the old format, and
// checks that migrating up gives every page the access it had before, and
// that migrating down again keeps it.
func TestMigrationConvertsPathRules(t *testing.T) {
	f := newApp(t)
	m := migration090(t)
	if err := m.Down(f.app); err != nil {
		t.Fatal(err)
	}

	for i, r := range []map[string]any{
		{"pattern": "/pub/**", "access": "public"},
		{"pattern": "/fin/**", "access": "restricted", "groups": []string{"finance", "ops"}},
		{"pattern": "/team/*", "access": "private"}, // one segment only
		{"pattern": "/**", "access": "private"},     // homepage only
	} {
		r["priority"] = i
		f.save(t, "access_rules", r)
	}
	cfg, err := f.app.FindFirstRecordByFilter("wiki_config", "")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Set("private_default", true)
	if err := f.app.Save(cfg); err != nil {
		t.Fatal(err)
	}
	alice := f.save(t, "users", map[string]any{"email": "alice@example.com", "password": "Passw0rd!Passw0rd", "role": "viewer", "groups": []string{"finance"}})
	want := map[string]string{
		"":         "private",
		"pub/a":    "public",
		"fin/a/b":  "restricted:finance,ops",
		"team/x":   "private",
		"team/x/y": "private", // unmatched → private_default
		"other":    "private",
	}
	for path := range want {
		f.save(t, "documents", map[string]any{"path": path, "title": path})
	}

	if err := m.Up(f.app); err != nil {
		t.Fatal(err)
	}
	groupName := map[string]string{}
	groups, err := f.app.FindAllRecords("groups")
	if err != nil {
		t.Fatal(err)
	}
	for _, g := range groups {
		groupName[g.Id] = g.GetString("name")
	}
	for path, w := range want {
		doc, err := f.app.FindFirstRecordByData("documents", "path", path)
		if err != nil {
			t.Fatal(path, err)
		}
		got := doc.GetString("access")
		if names := namesOf(doc.GetStringSlice("groups"), groupName); names != "" {
			got += ":" + names
		}
		if got != w {
			t.Errorf("after up, %q: %s, want %s", path, got, w)
		}
	}
	if got := namesOf(f.reload(t, "users", alice.Id).GetStringSlice("groups"), groupName); got != "finance" {
		t.Errorf("after up, alice's groups: %q, want finance", got)
	}
	if _, err := f.app.FindCollectionByNameOrId("access_rules"); err == nil {
		t.Error("access_rules still exists after up")
	}

	if err := m.Down(f.app); err != nil {
		t.Fatal(err)
	}
	rules, err := f.app.FindAllRecords("access_rules")
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != len(want) {
		t.Errorf("after down: %d access rules, want one per page (%d)", len(rules), len(want))
	}
	for _, r := range rules {
		got := r.GetString("access")
		if names := strings.Join(r.GetStringSlice("groups"), ","); names != "" {
			got += ":" + names
		}
		if path := strings.TrimPrefix(r.GetString("pattern"), "/"); got != want[path] {
			t.Errorf("after down, rule for %q: %s, want %s", path, got, want[path])
		}
	}
	if got := f.reload(t, "users", alice.Id).GetStringSlice("groups"); len(got) != 1 || got[0] != "finance" {
		t.Errorf("after down, alice's groups: %v, want [finance]", got)
	}
}

func namesOf(ids []string, groupName map[string]string) string {
	names := []string{}
	for _, id := range ids {
		names = append(names, groupName[id])
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}
