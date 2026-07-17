// Package service implements application business use-cases.
//
// Telegram handlers must call these services only. Services depend on
// injected repositories and the search engine, and never call Telegram APIs.
package service
