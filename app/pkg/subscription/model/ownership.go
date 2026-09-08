package model

import "strings"

// TagBelongsToSubscription identifies nodes by their persisted ownership suffix.
func TagBelongsToSubscription(tag, id string) bool { return strings.HasSuffix(tag, " | "+id) }
