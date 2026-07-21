package converter

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// LooksLikeJSONL reports whether path is likely newline-delimited JSON.
func LooksLikeJSONL(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".jsonl", ".ndjson":
		return true
	case ".json":
		// Single JSON array files are rare for dumps; still probe content.
	case ".txt", ".csv":
		// may still be JSONL misnamed
	default:
		if ext != "" && ext != ".json" {
			return false
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		return strings.HasPrefix(line, "{")
	}
	return false
}

// ConvertJSONLFile streams JSONL/NDJSON into id,name,phone,username,extras CSV.
//
// Preferred Telegram ID keys: adapterUserId, telegram_id, telegramId, user_id, uid.
// Bare "id" is kept in extras (often a CRM row id, not Telegram).
func ConvertJSONLFile(ctx context.Context, path string) (Result, error) {
	started := time.Now()
	in, err := os.Open(path)
	if err != nil {
		return Result{}, fmt.Errorf("converter: open jsonl %q: %w", path, err)
	}
	defer func() { _ = in.Close() }()

	outPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".standard.csv"
	out, err := os.Create(outPath)
	if err != nil {
		return Result{}, fmt.Errorf("converter: create %q: %w", outPath, err)
	}
	defer func() { _ = out.Close() }()

	cw := csv.NewWriter(out)
	if err := cw.Write([]string{"id", "name", "phone", "username", "extras"}); err != nil {
		return Result{}, err
	}

	sc := bufio.NewScanner(in)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	var inputRows, outputRows, skipped uint64
	for sc.Scan() {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		inputRows++

		var raw map[string]any
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			skipped++
			continue
		}

		id, name, phone, username, extras, ok := mapJSONLRecord(raw)
		if !ok {
			skipped++
			continue
		}
		if err := cw.Write([]string{id, name, phone, username, extras}); err != nil {
			return Result{}, err
		}
		outputRows++
	}
	if err := sc.Err(); err != nil && err != io.EOF {
		return Result{}, fmt.Errorf("converter: scan jsonl: %w", err)
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return Result{}, err
	}

	finished := time.Now()
	elapsed := finished.Sub(started).Seconds()
	rps := 0.0
	if elapsed > 0 {
		rps = float64(outputRows) / elapsed
	}
	return Result{
		InputFile:  path,
		OutputFile: outPath,
		Statistics: Statistics{
			InputRows:   inputRows,
			OutputRows:  outputRows,
			SkippedRows: skipped,
			StartedAt:   started,
			FinishedAt:  finished,
			RowsPerSec:  rps,
		},
	}, nil
}

func mapJSONLRecord(raw map[string]any) (id, name, phone, username, extras string, ok bool) {
	id = firstJSONString(raw,
		"adapterUserId", "adapter_user_id", "adapteruserid",
		"telegram_id", "telegramId", "telegramid", "telegramUserId", "telegram_user_id",
		"user_id", "userId", "userid", "uid",
	)
	if id == "" || !jsonlValidID(id) {
		return "", "", "", "", "", false
	}

	name = firstJSONString(raw, "name", "fullName", "full_name")
	if name == "" {
		name = strings.TrimSpace(firstJSONString(raw, "firstName", "first_name") + " " + firstJSONString(raw, "lastName", "last_name"))
	}
	if name == "" {
		name = firstJSONString(raw, "nick", "nickname", "displayName", "display_name")
	}

	phone = firstJSONString(raw, "phone", "phoneNumber", "phone_number", "mobile")

	nick := firstJSONString(raw, "nick", "nickname", "username", "userName", "login")
	if looksLikeTelegramUsername(nick) {
		username = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(nick), "@"))
	}

	extrasMap := make(map[string]any, len(raw))
	for k, v := range raw {
		nk := normalizeJSONKey(k)
		switch nk {
		case "adapteruserid", "adapter_user_id",
			"telegram_id", "telegramid", "telegramuserid", "telegram_user_id",
			"user_id", "userid", "uid",
			"name", "fullname", "full_name",
			"firstname", "first_name", "lastname", "last_name",
			"phone", "phonenumber", "phone_number", "mobile":
			continue
		case "nick", "nickname", "username", "login":
			if username != "" {
				continue // already used as username
			}
		}
		extrasMap[k] = v
	}
	extras = "{}"
	if len(extrasMap) > 0 {
		b, err := json.Marshal(extrasMap)
		if err == nil {
			extras = string(b)
		}
	}
	return id, name, phone, username, extras, true
}

func firstJSONString(raw map[string]any, keys ...string) string {
	// Exact key match first.
	for _, key := range keys {
		if v, ok := raw[key]; ok {
			if s := stringifyJSON(v); s != "" {
				return s
			}
		}
	}
	// Normalized key match.
	normWant := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		normWant[normalizeJSONKey(key)] = struct{}{}
	}
	for k, v := range raw {
		if _, ok := normWant[normalizeJSONKey(k)]; !ok {
			continue
		}
		if s := stringifyJSON(v); s != "" {
			return s
		}
	}
	return ""
}

func normalizeJSONKey(s string) string {
	var b strings.Builder
	for _, r := range strings.TrimSpace(s) {
		if unicode.IsSpace(r) || r == '_' || r == '-' {
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

func stringifyJSON(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case json.Number:
		return strings.TrimSpace(t.String())
	case float64:
		// JSON numbers decode as float64 by default.
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strings.TrimSpace(strconv.FormatFloat(t, 'f', -1, 64))
	case bool:
		if t {
			return "true"
		}
		return "false"
	default:
		b, err := json.Marshal(t)
		if err != nil {
			return strings.TrimSpace(fmt.Sprint(t))
		}
		s := strings.TrimSpace(string(b))
		return strings.Trim(s, `"`)
	}
}

func jsonlValidID(id string) bool {
	if id == "" || len(id) > 32 {
		return false
	}
	for _, r := range id {
		if r < '0' || r > '9' {
			return false
		}
	}
	return id[0] != '0' || id == "0"
}

func looksLikeTelegramUsername(raw string) bool {
	u := strings.ToLower(strings.TrimSpace(raw))
	u = strings.TrimPrefix(u, "@")
	if len(u) < 3 || len(u) > 64 {
		return false
	}
	for i, r := range u {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '_':
		default:
			return false
		}
		if i == 0 && r >= '0' && r <= '9' {
			return false
		}
	}
	return true
}
