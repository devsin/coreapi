// Package reserved provides a single source of truth for reserved path
// segments.  Both the URL-shortener (short codes) and the user service
// (usernames) check against this list so that no user-generated identifier
// can shadow an application route or well-known path.
package reserved

import "strings"

// IsReserved returns true when code (case-insensitive) matches a reserved
// word.  Use this for validating both short link codes and usernames.
func IsReserved(code string) bool {
	return words[strings.ToLower(code)]
}

// words is the merged set of every path segment the platform reserves.
var words = map[string]bool{
	// Current SvelteKit routes
	"login": true, "signup": true, "auth": true, "logout": true,
	"links": true, "insights": true, "settings": true,
	"explore": true, "search": true, "contact": true,
	"about": true, "privacy": true, "terms": true,

	// API & infrastructure
	"api": true, "health": true, "healthz": true, "ready": true,
	".well-known": true, "robots.txt": true, "sitemap.xml": true,
	"favicon.ico": true,

	// Future routes
	"admin": true, "dashboard": true, "notifications": true, "messages": true,
	"help": true, "faq": true, "pricing": true, "blog": true, "docs": true,
	"embed": true, "import": true, "export": true, "report": true,
	"invite": true, "verify": true, "unsubscribe": true, "subscribe": true,
	"download": true, "upload": true, "share": true, "callback": true,
	"feed": true, "rss": true, "atom": true, "webhooks": true,
	"billing": true, "checkout": true, "account": true, "profile": true,
	"onboarding": true, "welcome": true, "getting-started": true,
	"changelog": true, "releases": true, "status": true,
	"developers": true, "partners": true, "affiliates": true,
	"careers": true, "jobs": true, "press": true, "media": true,
	"legal": true, "security": true, "compliance": true, "dmca": true,
	"support": true, "guidelines": true, "brand": true, "assets": true,
	"app": true, "mobile": true, "desktop": true,
	"404": true, "500": true, "error": true,

	// Auth / security / account flows
	"oauth": true, "oidc": true, "sso": true, "saml": true,
	"mfa": true, "2fa": true, "totp": true, "passkeys": true,
	"reset": true, "forgot": true, "password": true, "password-reset": true,
	"confirm": true, "confirm-email": true, "email-confirmation": true,
	"session": true, "sessions": true, "token": true, "tokens": true, "jwt": true,
	"consent": true,

	// Platform / integration endpoints
	"graphql": true,
	"metrics": true, "ping": true,
	"openapi": true, "swagger": true, "redoc": true,
	"version": true, "v1": true, "v2": true, "v3": true,
	"static": true, "public": true,
	"cdn": true, "img": true, "images": true, "media-assets": true, "uploads": true,

	// Framework-reserved paths
	"_next": true, "_nuxt": true, "_astro": true, "_svelte": true, "_app": true, "_data": true,

	// Well-known / verification / trust files
	"apple-app-site-association": true,
	"assetlinks.json":            true,
	"security.txt":               true,
	"humans.txt":                 true,
	"ads.txt":                    true,
	"gpc.json":                   true,

	// Admin / internal / ops
	"internal": true, "private": true, "staff": true,
	"moderation": true, "mod": true,
	"ops": true, "ops-tools": true,
	"maintenance": true,
	"queue":       true, "workers": true,

	// Teams / orgs / identities
	"me":   true,
	"user": true, "users": true,
	"team": true, "teams": true,
	"org": true, "orgs": true, "organizations": true,
	"projects": true, "workspaces": true,
	"collection": true, "collections": true,

	// Redirect-ish common routes
	"short": true, "go": true, "redirect": true,

	// Payments / commerce
	"payments": true, "payment": true,
	"invoices": true, "invoice": true,
	"refunds":       true,
	"subscriptions": true,
	"stripe":        true,

	// Trust / safety / abuse
	"abuse": true,
	"trust": true, "trust-safety": true,
	"takedown":     true,
	"appeal":       true,
	"transparency": true,

	// i18n / misc
	"i18n": true, "locale": true, "locales": true,
	"robots": true, "sitemap": true,
	"connect": true, "return": true, "redirect-uri": true,

	// Brand names
	"core": true, "official": true, "root": true, "system": true, "www": true, "web": true,

	// All single-letter codes (a-z)
	"a": true, "b": true, "c": true, "d": true, "e": true, "f": true, "g": true,
	"h": true, "i": true, "j": true, "k": true, "l": true, "m": true, "n": true,
	"o": true, "p": true, "q": true, "r": true, "s": true, "t": true, "u": true,
	"v": true, "w": true, "x": true, "y": true, "z": true,
}
