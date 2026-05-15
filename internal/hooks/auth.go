package hooks

import (
	"strings"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
)

// registerAuthHooks installs the OAuth email-domain allowlist on the `users`
// collection. This is the headline ergonomic win over running oauth2-proxy in
// front of the app: a couple of lines here replace a separate proxy service.
//
// The allowlist lives in wiki_config.oauth_email_allowlist. Each entry is
// either a full email ("alice@example.com") or a bare domain ("example.com")
// — the latter matches anything ending in "@<domain>". An empty allowlist
// disables the check (any email is accepted).
func registerAuthHooks(app *pocketbase.PocketBase) {
	// Default role for any newly-created user, since `role` is Required on
	// the collection but OAuth sign-up doesn't supply one. Admins can promote
	// later via the Users admin page.
	app.OnRecordCreate("users").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("role") == "" {
			e.Record.Set("role", "viewer")
		}
		return e.Next()
	})

	app.OnRecordAuthWithOAuth2Request("users").BindFunc(func(e *core.RecordAuthWithOAuth2RequestEvent) error {
		if e.OAuth2User == nil || e.OAuth2User.Email == "" {
			return e.Next()
		}
		allowlist, err := loadOAuthAllowlist(e.App)
		if err != nil {
			return err
		}
		if len(allowlist) == 0 {
			return e.Next()
		}
		if !emailAllowed(e.OAuth2User.Email, allowlist) {
			return apis.NewForbiddenError("Email domain is not on the allow-list.", nil)
		}
		return e.Next()
	})
}

func loadOAuthAllowlist(app core.App) ([]string, error) {
	cfg, err := app.FindFirstRecordByFilter("wiki_config", "")
	if err != nil {
		return nil, err
	}
	return cfg.GetStringSlice("oauth_email_allowlist"), nil
}

func emailAllowed(email string, allowlist []string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	for _, entry := range allowlist {
		entry = strings.ToLower(strings.TrimSpace(entry))
		if entry == "" {
			continue
		}
		if strings.Contains(entry, "@") {
			if email == entry {
				return true
			}
			continue
		}
		// Bare domain: match the suffix after "@".
		if strings.HasSuffix(email, "@"+entry) {
			return true
		}
	}
	return false
}
