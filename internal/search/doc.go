// Package search provides an exact-lookup search engine over LMDB indexes.
//
// Lookups are performed by ID, phone, or username only. Phone and username
// queries resolve through their index to an ID, then load the full record from
// the ID database. Substring and fuzzy matching are intentionally unsupported.
package search
