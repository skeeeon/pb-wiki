package migrations

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Replace path-glob access rules with access attributes on each document,
// enforced by the documents collection's API rules.
//
// Before, documents had ListRule = ViewRule = "" and hooks checked path rules
// after the query. That leaked private documents through list totals, page
// boundaries and realtime broadcasts, which only check collection rules.
// Now every document carries `access` (public/private/restricted) and
// `groups`, and one native rule checks them inside SQL for list, view and
// realtime alike.
//
// Groups become a collection so the group check is a plain relation
// comparison (`groups.id ?= @request.auth.groups.id`: at least one group in
// common). users.groups and access_rules.groups were JSON lists of names.
//
// Each document's access is computed once from the old access_rules, using a
// frozen copy of the old glob matcher, and access_rules is dropped.
// Migrating down turns every document's attributes back into an exact-path
// access rule, so nobody's access changes in either direction.
//
// The users rules also stop sign-up and self-edit from setting role or
// groups; before this migration anyone could create an admin account.

const (
	writerRule = `@request.auth.role = "admin" || @request.auth.role = "editor"`
	adminRule  = `@request.auth.role = "admin"`
	readRule   = `@request.auth.role = "admin" ||
(access = "public" && (@request.auth.id != "" || @collection.wiki_config.require_login = false)) ||
(access = "private" && @request.auth.id != "") ||
(access = "restricted" && groups.id ?= @request.auth.groups.id)`
)

func init() {
	m.Register(func(app core.App) error {
		// Read everything that still uses group names before the schema changes.
		userGroups, err := readNameLists(app, "users")
		if err != nil {
			return err
		}
		rules, err := readLegacyRules(app)
		if err != nil {
			return err
		}
		var cfg struct {
			PrivateDefault bool `db:"private_default"`
		}
		if err := app.DB().NewQuery("SELECT [[private_default]] FROM {{wiki_config}} LIMIT 1").One(&cfg); err != nil {
			return err
		}

		groups := core.NewBaseCollection("groups")
		groups.Fields.Add(&core.TextField{Name: "name", Required: true, Max: 100})
		groups.Indexes = []string{"CREATE UNIQUE INDEX `idx_groups_name` ON `groups` (`name`)"}
		groups.ListRule = ptr(writerRule)
		groups.ViewRule = ptr(writerRule)
		groups.CreateRule = ptr(adminRule)
		groups.UpdateRule = ptr(adminRule)
		groups.DeleteRule = ptr(adminRule)
		if err := app.Save(groups); err != nil {
			return err
		}
		groupID := map[string]string{}
		ids := func(names []string) ([]string, error) {
			out := []string{}
			for _, name := range names {
				if groupID[name] == "" {
					g := core.NewRecord(groups)
					g.Set("name", name)
					if err := app.Save(g); err != nil {
						return nil, err
					}
					groupID[name] = g.Id
				}
				out = append(out, groupID[name])
			}
			return out, nil
		}

		// users.groups: JSON names → relation.
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		users.Fields.RemoveByName("groups")
		if err := app.Save(users); err != nil {
			return err
		}
		users.Fields.Add(&core.RelationField{Name: "groups", CollectionId: groups.Id, MaxSelect: 100})
		if err := app.Save(users); err != nil {
			return err
		}
		for userID, names := range userGroups {
			groupIDs, err := ids(names)
			if err != nil {
				return err
			}
			if err := setJSON(app, "users", userID, "groups", groupIDs); err != nil {
				return err
			}
		}

		// documents: add access + groups, computed from the old rules.
		docs, err := app.FindCollectionByNameOrId("documents")
		if err != nil {
			return err
		}
		docs.Fields.Add(&core.SelectField{
			Name:      "access",
			Required:  true,
			MaxSelect: 1,
			Values:    []string{"public", "private", "restricted"},
		})
		docs.Fields.Add(&core.RelationField{Name: "groups", CollectionId: groups.Id, MaxSelect: 100})
		if err := app.Save(docs); err != nil {
			return err
		}
		var paths []struct {
			ID   string `db:"id"`
			Path string `db:"path"`
		}
		if err := app.DB().NewQuery("SELECT [[id]], [[path]] FROM {{documents}}").All(&paths); err != nil {
			return err
		}
		for _, d := range paths {
			level, names := legacyAccess(d.Path, rules, cfg.PrivateDefault)
			groupIDs, err := ids(names)
			if err != nil {
				return err
			}
			if err := setJSON(app, "documents", d.ID, "groups", groupIDs); err != nil {
				return err
			}
			_, err = app.DB().Update("documents", dbx.Params{"access": level}, dbx.HashExp{"id": d.ID}).Execute()
			if err != nil {
				return err
			}
		}

		old, err := app.FindCollectionByNameOrId("access_rules")
		if err != nil {
			return err
		}
		if err := app.Delete(old); err != nil {
			return err
		}

		docs.ListRule = ptr(readRule)
		docs.ViewRule = ptr(readRule)
		docs.CreateRule = ptr(writerRule)
		docs.UpdateRule = ptr("(" + writerRule + ") && (" + readRule + ")")
		docs.DeleteRule = ptr("(" + writerRule + ") && (" + readRule + ")")
		if err := app.Save(docs); err != nil {
			return err
		}

		users.CreateRule = ptr(adminRule + ` || (@request.body.role:isset = false && @request.body.groups:isset = false)`)
		users.UpdateRule = ptr(adminRule + ` || (id = @request.auth.id && @request.body.role:changed = false && @request.body.groups:changed = false)`)
		return app.Save(users)
	}, func(app core.App) error {
		groupName := map[string]string{}
		var groupRows []struct {
			ID   string `db:"id"`
			Name string `db:"name"`
		}
		if err := app.DB().NewQuery("SELECT [[id]], [[name]] FROM {{groups}}").All(&groupRows); err != nil {
			return err
		}
		for _, g := range groupRows {
			groupName[g.ID] = g.Name
		}
		names := func(ids []string) []string {
			out := []string{}
			for _, id := range ids {
				if groupName[id] != "" {
					out = append(out, groupName[id])
				}
			}
			return out
		}
		userGroups, err := readNameLists(app, "users")
		if err != nil {
			return err
		}
		docGroups, err := readNameLists(app, "documents")
		if err != nil {
			return err
		}
		var docRows []struct {
			ID     string `db:"id"`
			Path   string `db:"path"`
			Access string `db:"access"`
		}
		if err := app.DB().NewQuery("SELECT [[id]], [[path]], [[access]] FROM {{documents}}").All(&docRows); err != nil {
			return err
		}

		// One exact-path rule per document keeps everyone's access unchanged.
		rules := core.NewBaseCollection("access_rules")
		rules.Fields.Add(&core.TextField{Name: "pattern", Required: true, Max: 500})
		rules.Fields.Add(&core.SelectField{Name: "access", Required: true, MaxSelect: 1, Values: []string{"public", "private", "restricted"}})
		rules.Fields.Add(&core.JSONField{Name: "groups"})
		rules.Fields.Add(&core.NumberField{Name: "priority"})
		rules.Fields.Add(&core.TextField{Name: "description", Max: 500})
		rules.Fields.Add(&core.AutodateField{Name: "created", OnCreate: true})
		rules.Fields.Add(&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true})
		rules.Indexes = []string{"CREATE INDEX `idx_access_rules_priority` ON `access_rules` (`priority`)"}
		rules.ListRule = ptr(adminRule)
		rules.ViewRule = ptr(adminRule)
		rules.CreateRule = ptr(adminRule)
		rules.UpdateRule = ptr(adminRule)
		rules.DeleteRule = ptr(adminRule)
		if err := app.Save(rules); err != nil {
			return err
		}
		for _, d := range docRows {
			r := core.NewRecord(rules)
			r.Set("pattern", "/"+d.Path)
			r.Set("access", d.Access)
			r.Set("groups", names(docGroups[d.ID]))
			r.Set("description", "migrated from document access")
			if err := app.Save(r); err != nil {
				return err
			}
		}

		docs, err := app.FindCollectionByNameOrId("documents")
		if err != nil {
			return err
		}
		docs.ListRule = ptr("")
		docs.ViewRule = ptr("")
		docs.CreateRule = ptr(`@request.auth.id != "" && (` + writerRule + `)`)
		docs.UpdateRule = docs.CreateRule
		docs.DeleteRule = docs.CreateRule
		docs.Fields.RemoveByName("access")
		docs.Fields.RemoveByName("groups")
		if err := app.Save(docs); err != nil {
			return err
		}

		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		// The rules mention groups, so they change in the same save that drops it.
		users.CreateRule = ptr("")
		users.UpdateRule = ptr(`id = @request.auth.id || @request.auth.role = "admin"`)
		users.Fields.RemoveByName("groups")
		if err := app.Save(users); err != nil {
			return err
		}
		users.Fields.Add(&core.JSONField{Name: "groups"})
		if err := app.Save(users); err != nil {
			return err
		}
		for userID, ids := range userGroups {
			if err := setJSON(app, "users", userID, "groups", names(ids)); err != nil {
				return err
			}
		}

		groups, err := app.FindCollectionByNameOrId("groups")
		if err != nil {
			return err
		}
		return app.Delete(groups)
	})
}

func ptr(s string) *string { return &s }

// readNameLists returns the `groups` JSON array of every row in table, keyed
// by record id. Rows with no groups are omitted.
func readNameLists(app core.App, table string) (map[string][]string, error) {
	var rows []struct {
		ID     string `db:"id"`
		Groups string `db:"groups"`
	}
	err := app.DB().NewQuery("SELECT [[id]], COALESCE([[groups]], '') AS [[groups]] FROM {{" + table + "}}").All(&rows)
	if err != nil {
		return nil, err
	}
	out := map[string][]string{}
	for _, r := range rows {
		var list []string
		if json.Unmarshal([]byte(r.Groups), &list) == nil && len(list) > 0 {
			out[r.ID] = list
		}
	}
	return out, nil
}

func setJSON(app core.App, table, id, column string, value []string) error {
	b, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = app.DB().Update(table, dbx.Params{column: string(b)}, dbx.HashExp{"id": id}).Execute()
	return err
}

// --- frozen copy of the old path-rule evaluator (internal/access) ----------
// Used only to convert existing rules. Do not change: it must reproduce the
// behaviour documents had before this migration.

type legacyRule struct {
	Pattern string
	Access  string
	Groups  []string
}

func readLegacyRules(app core.App) ([]legacyRule, error) {
	var rows []struct {
		Pattern string `db:"pattern"`
		Access  string `db:"access"`
		Groups  string `db:"groups"`
	}
	err := app.DB().NewQuery(
		"SELECT [[pattern]], [[access]], COALESCE([[groups]], '') AS [[groups]] FROM {{access_rules}} ORDER BY [[priority]] ASC, rowid ASC",
	).All(&rows)
	if err != nil {
		return nil, err
	}
	rules := make([]legacyRule, len(rows))
	for i, r := range rows {
		rules[i] = legacyRule{Pattern: r.Pattern, Access: r.Access}
		_ = json.Unmarshal([]byte(r.Groups), &rules[i].Groups)
	}
	return rules, nil
}

// legacyAccess returns the access a document had under the old rules: the
// first matching rule wins, and unmatched paths follow private_default. An
// unknown access level used to deny everyone but admins, which restricted
// with no groups reproduces.
func legacyAccess(path string, rules []legacyRule, privateDefault bool) (string, []string) {
	for _, r := range rules {
		if !legacyMatch(r.Pattern, path) {
			continue
		}
		switch r.Access {
		case "public", "private":
			return r.Access, nil
		case "restricted":
			return "restricted", r.Groups
		default:
			return "restricted", nil
		}
	}
	if privateDefault {
		return "private", nil
	}
	return "public", nil
}

func legacyMatch(pattern, path string) bool {
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if !strings.HasPrefix(pattern, "/") {
		pattern = "/" + pattern
	}
	if pattern == "/**" {
		return path == "/"
	}
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		if i+3 <= len(pattern) && pattern[i:i+3] == "/**" && i+3 == len(pattern) {
			b.WriteString("(/.*)?")
			break
		}
		if strings.HasPrefix(pattern[i:], "**") {
			b.WriteString(".*")
			i++
		} else if pattern[i] == '*' {
			b.WriteString("[^/]*")
		} else if pattern[i] == '?' {
			b.WriteString("[^/]")
		} else {
			b.WriteString(regexp.QuoteMeta(string(pattern[i])))
		}
	}
	b.WriteString("$")
	matched, _ := regexp.MatchString(b.String(), path)
	return matched
}
