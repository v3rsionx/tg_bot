package converter

import "time"

// Field role for mapped columns.
type FieldRole string

const (
	RoleID       FieldRole = "id"
	RoleName     FieldRole = "name"
	RoleLastName FieldRole = "lastname"
	RolePhone    FieldRole = "phone"
	RoleUsername FieldRole = "username"
	RoleExtras   FieldRole = "extras"
)

// EncodingName identifies a supported input encoding.
type EncodingName string

const (
	EncodingUTF8       EncodingName = "UTF-8"
	EncodingUTF8BOM    EncodingName = "UTF-8-BOM"
	EncodingUTF16LE    EncodingName = "UTF-16LE"
	EncodingUTF16BE    EncodingName = "UTF-16BE"
	EncodingWindows1251 EncodingName = "Windows-1251"
	EncodingLatin1     EncodingName = "Latin1"
)

// DelimiterName is a human-readable delimiter label.
type DelimiterName string

const (
	DelimiterComma     DelimiterName = "comma"
	DelimiterSemicolon DelimiterName = "semicolon"
	DelimiterPipe      DelimiterName = "pipe"
	DelimiterTab       DelimiterName = "tab"
)

// ColumnMapping describes how source headers map to the standard schema.
type ColumnMapping struct {
	IDIndex       int      // -1 if missing
	NameIndex     int
	LastNameIndex int
	PhoneIndex    int
	UsernameIndex int
	ExtrasIndexes []int    // unknown columns preserved in extras JSON
	ExtrasNames   []string // original header names for extras columns
	Headers       []string
}

// Detection holds autodetection results for a source file.
type Detection struct {
	Encoding  EncodingName
	Delimiter rune
	DelimiterName DelimiterName
	HasHeader bool
	Mapping   ColumnMapping
}

// Statistics tracks conversion counters.
type Statistics struct {
	InputRows    uint64
	OutputRows   uint64
	SkippedRows  uint64
	BytesRead    int64
	BytesTotal   int64
	StartedAt    time.Time
	FinishedAt   time.Time
	RowsPerSec   float64
}

// Progress is a point-in-time progress snapshot.
type Progress struct {
	File         string
	Processed    uint64
	Output       uint64
	Skipped      uint64
	Percent      float64
	RowsPerSec   float64
	ETA          time.Duration
	BytesRead    int64
	BytesTotal   int64
}

// ProgressFunc receives continuous progress updates.
type ProgressFunc func(Progress)

// DryRunReport summarizes detection using only the first N rows.
type DryRunReport struct {
	File       string
	Detection  Detection
	SampleRows int
	ExtrasKeys []string
}

// Result is returned after a successful conversion.
type Result struct {
	InputFile  string
	OutputFile string
	Statistics Statistics
	Detection  Detection
}
