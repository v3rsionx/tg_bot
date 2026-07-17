package validator

import (
	"strconv"
	"strings"
)

// OwnerIDs validates a list of owner Telegram user IDs.
func (v *Standard) OwnerIDs(ids []int64) error {
	if len(ids) == 0 {
		return Error{Field: "BOT_OWNER_IDS", Message: "must contain at least one owner ID"}
	}
	if len(ids) > 100 {
		return Error{Field: "BOT_OWNER_IDS", Message: "must contain at most 100 owner IDs"}
	}
	seen := make(map[int64]struct{}, len(ids))
	for _, id := range ids {
		if err := v.TelegramUserID(id); err != nil {
			return Error{Field: "BOT_OWNER_IDS", Message: "contains an invalid Telegram user ID"}
		}
		if _, exists := seen[id]; exists {
			return Error{Field: "BOT_OWNER_IDS", Message: "must not contain duplicate IDs"}
		}
		seen[id] = struct{}{}
	}
	return nil
}

// OwnerIDsCSV validates a comma-separated owner ID list.
func (v *Standard) OwnerIDsCSV(value string) ([]int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, Error{Field: "BOT_OWNER_IDS", Message: "is required"}
	}
	parts := strings.Split(value, ",")
	ids := make([]int64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, Error{Field: "BOT_OWNER_IDS", Message: "must be comma-separated positive integers"}
		}
		id, err := strconv.ParseInt(part, 10, 64)
		if err != nil {
			return nil, Error{Field: "BOT_OWNER_IDS", Message: "must be comma-separated positive integers"}
		}
		ids = append(ids, id)
	}
	if err := v.OwnerIDs(ids); err != nil {
		return nil, err
	}
	return ids, nil
}
