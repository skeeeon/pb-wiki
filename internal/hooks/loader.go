package hooks

import (
	"github.com/pocketbase/pocketbase/core"

	"github.com/skeeeon/pb-wiki/internal/access"
)

// loadRules returns all access_rules ordered by ascending priority (first
// matching rule wins). Limit 0 means no SQL LIMIT clause — we fetch all rules
// because the evaluator walks them in order; rule counts are expected to be
// small (tens at most for a real wiki).
func loadRules(app core.App) ([]access.Rule, error) {
	records, err := app.FindRecordsByFilter("access_rules", "", "+priority", 0, 0)
	if err != nil {
		return nil, err
	}
	rules := make([]access.Rule, len(records))
	for i, r := range records {
		rules[i] = access.Rule{
			Pattern: r.GetString("pattern"),
			Access:  r.GetString("access"),
			Groups:  r.GetStringSlice("groups"),
		}
	}
	return rules, nil
}

// configFlags is the bundle of wiki_config booleans the access evaluator
// needs. Loaded together because each document hook needs both — saves a
// duplicate FindFirstRecordByFilter call per request.
type configFlags struct {
	privateDefault bool
	requireLogin   bool
}

// loadConfigFlags returns the access-relevant flags from wiki_config.
func loadConfigFlags(app core.App) (configFlags, error) {
	cfg, err := app.FindFirstRecordByFilter("wiki_config", "")
	if err != nil {
		return configFlags{}, err
	}
	return configFlags{
		privateDefault: cfg.GetBool("private_default"),
		requireLogin:   cfg.GetBool("require_login"),
	}, nil
}

// recordToUser projects an auth record onto the minimal shape the access
// evaluator needs. Returns nil for anonymous requests.
func recordToUser(r *core.Record) *access.User {
	if r == nil {
		return nil
	}
	return &access.User{
		Role:   r.GetString("role"),
		Groups: r.GetStringSlice("groups"),
	}
}
