package converter

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Converter streams dump files into standard CSV output.
type Converter struct {
	cfg         Config
	log         Logger
	progressFn  ProgressFunc
	skipLog     *SkipLogger
	checkpoints *CheckpointStore
}

// New constructs a Converter with dependency-injected collaborators.
func New(cfg Config, log Logger, progressFn ProgressFunc) (*Converter, error) {
	cfg = cfg.withDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if log == nil {
		log = NopLogger{}
	}
	skip, err := NewSkipLogger(cfg.LogPath)
	if err != nil {
		return nil, err
	}
	store := NewCheckpointStore(filepath.Join(cfg.CheckpointDir, "converter.checkpoint.json"))
	if cfg.Resume {
		if err := store.Load(); err != nil {
			_ = skip.Close()
			return nil, err
		}
	}
	return &Converter{
		cfg:         cfg,
		log:         log,
		progressFn:  progressFn,
		skipLog:     skip,
		checkpoints: store,
	}, nil
}

// Close releases converter resources.
func (c *Converter) Close() error {
	if c == nil || c.skipLog == nil {
		return nil
	}
	return c.skipLog.Close()
}

// DryRun detects encoding/delimiter/mapping using the first N rows.
func (c *Converter) DryRun(ctx context.Context, path string) (DryRunReport, error) {
	det, sampleRows, err := c.detectFile(ctx, path, c.cfg.DryRunRows)
	if err != nil {
		return DryRunReport{}, err
	}
	return DryRunReport{
		File:       path,
		Detection:  det,
		SampleRows: sampleRows,
		ExtrasKeys: append([]string(nil), det.Mapping.ExtrasNames...),
	}, nil
}

// Run converts all configured sources (or dry-runs them).
func (c *Converter) Run(ctx context.Context) ([]Result, []DryRunReport, error) {
	if c.cfg.DryRun {
		reports := make([]DryRunReport, 0, len(c.cfg.Sources))
		for _, src := range c.cfg.Sources {
			if err := ctx.Err(); err != nil {
				return nil, reports, err
			}
			rep, err := c.DryRun(ctx, src)
			if err != nil {
				return nil, reports, err
			}
			reports = append(reports, rep)
		}
		return nil, reports, nil
	}

	results := make([]Result, 0, len(c.cfg.Sources))
	for _, src := range c.cfg.Sources {
		if err := ctx.Err(); err != nil {
			return results, nil, err
		}
		res, err := c.ConvertFile(ctx, src)
		if err != nil {
			return results, nil, err
		}
		results = append(results, res)
	}
	return results, nil, nil
}

// ConvertFile converts one source file to filename.standard.csv.
func (c *Converter) ConvertFile(ctx context.Context, path string) (Result, error) {
	det, _, err := c.detectFile(ctx, path, 50)
	if err != nil {
		return Result{}, err
	}
	outPath := standardOutputPath(path)

	stats := newStats()
	info, err := os.Stat(path)
	if err != nil {
		return Result{}, wrap("stat input", err)
	}
	stats.bytesTotal.Store(info.Size())

	var resumeLines uint64
	var outRows, skipped, inRows uint64
	headerDone := false
	if c.cfg.Resume {
		if cp, ok := c.checkpoints.Get(path); ok {
			resumeLines = cp.LineNumber
			inRows = cp.InputRows
			outRows = cp.OutputRows
			skipped = cp.SkippedRows
			headerDone = cp.HeaderDone
			stats.inputRows.Store(inRows)
			stats.outputRows.Store(outRows)
			stats.skippedRows.Store(skipped)
			c.log.Infof("resume %s after %d lines", path, resumeLines)
		}
	}

	in, err := os.Open(path)
	if err != nil {
		return Result{}, wrap("open input", err)
	}
	defer in.Close()

	counter := &countingReader{r: in}
	dec, err := newDecodingReader(counter, det.Encoding)
	if err != nil {
		return Result{}, err
	}
	reader := bufio.NewReaderSize(dec, c.cfg.ReadBufferBytes)

	outFlags := os.O_CREATE | os.O_WRONLY
	if headerDone && resumeLines > 0 {
		outFlags |= os.O_APPEND
	} else {
		outFlags |= os.O_TRUNC
		headerDone = false
	}
	out, err := os.OpenFile(outPath, outFlags, 0o644)
	if err != nil {
		return Result{}, wrap("open output", err)
	}
	defer out.Close()

	cw := csv.NewWriter(bufio.NewWriterSize(out, c.cfg.ReadBufferBytes))
	cw.Comma = ','
	if !headerDone {
		if err := cw.Write([]string{"id", "name", "phone", "username", "extras"}); err != nil {
			return Result{}, wrap("write header", err)
		}
		cw.Flush()
		headerDone = true
	}

	var lineNo uint64
	lastProgress := time.Now()
	lastCheckpoint := time.Now()

	for {
		if err := ctx.Err(); err != nil {
			_ = c.saveCheckpoint(path, outPath, counter.n, lineNo, stats, headerDone, det)
			return Result{}, err
		}

		line, err := readLine(reader)
		if err == io.EOF {
			if line == "" {
				break
			}
		} else if err != nil {
			return Result{}, wrap("read line", err)
		}

		lineNo++
		stats.bytesRead.Store(counter.n)

		if resumeLines > 0 && lineNo <= resumeLines {
			if err == io.EOF {
				break
			}
			continue
		}

		if strings.TrimSpace(line) == "" {
			if err == io.EOF {
				break
			}
			continue
		}

		// Skip header row once when present.
		if det.HasHeader && lineNo == 1 && resumeLines == 0 {
			if err == io.EOF {
				break
			}
			continue
		}

		stats.inputRows.Add(1)
		fields := parseCSVLine(line, det.Delimiter)
		id := fieldAt(fields, det.Mapping.IDIndex)
		if id == "" {
			stats.skippedRows.Add(1)
			_ = c.skipLog.LogSkip(path, lineNo, "missing id")
		} else {
			name := combineName(
				fieldAt(fields, det.Mapping.NameIndex),
				fieldAt(fields, det.Mapping.LastNameIndex),
			)
			phone := fieldAt(fields, det.Mapping.PhoneIndex)
			username := strings.TrimPrefix(fieldAt(fields, det.Mapping.UsernameIndex), "@")
			extras := buildExtrasJSON(fields, det.Mapping)
			if werr := cw.Write([]string{id, name, phone, username, extras}); werr != nil {
				return Result{}, wrap("write row", werr)
			}
			stats.outputRows.Add(1)
		}

		now := time.Now()
		if c.progressFn != nil && now.Sub(lastProgress) >= c.cfg.ProgressInterval {
			lastProgress = now
			c.progressFn(buildProgress(path, stats.snapshot()))
		}
		if c.cfg.Resume && now.Sub(lastCheckpoint) >= 2*time.Second {
			lastCheckpoint = now
			cw.Flush()
			_ = c.saveCheckpoint(path, outPath, counter.n, lineNo, stats, headerDone, det)
		}
		if err == io.EOF {
			break
		}
	}

	cw.Flush()
	if err := cw.Error(); err != nil {
		return Result{}, wrap("flush csv", err)
	}
	stats.bytesRead.Store(counter.n)
	stats.finish()
	if c.progressFn != nil {
		c.progressFn(buildProgress(path, stats.snapshot()))
	}
	if c.cfg.Resume {
		_ = c.checkpoints.Clear(path)
	}

	st := stats.snapshot()
	c.log.Infof("converted %s -> %s (%s)", path, outPath, FormatProgress(buildProgress(path, st)))
	return Result{
		InputFile:  path,
		OutputFile: outPath,
		Statistics: st,
		Detection:  det,
	}, nil
}

func (c *Converter) saveCheckpoint(path, outPath string, offset int64, line uint64, stats *statsAccumulator, headerDone bool, det Detection) error {
	st := stats.snapshot()
	c.checkpoints.Set(FileCheckpoint{
		InputFile:   path,
		OutputFile:  outPath,
		ByteOffset:  offset,
		LineNumber:  line,
		InputRows:   st.InputRows,
		OutputRows:  st.OutputRows,
		SkippedRows: st.SkippedRows,
		HeaderDone:  headerDone,
		Encoding:    string(det.Encoding),
		Delimiter:   string(det.Delimiter),
	})
	return c.checkpoints.Save()
}

func (c *Converter) detectFile(ctx context.Context, path string, sampleRows int) (Detection, int, error) {
	_ = ctx
	raw, err := peekFile(path, 64<<10)
	if err != nil {
		return Detection{}, 0, err
	}
	enc := c.cfg.ForceEncoding
	if enc == "" {
		enc = detectEncoding(raw)
	}

	f, err := os.Open(path)
	if err != nil {
		return Detection{}, 0, wrap("open for detect", err)
	}
	defer f.Close()

	dec, err := newDecodingReader(f, enc)
	if err != nil {
		return Detection{}, 0, err
	}
	br := bufio.NewReaderSize(dec, c.cfg.ReadBufferBytes)

	var sample strings.Builder
	lines := make([]string, 0, sampleRows)
	for len(lines) < sampleRows {
		line, err := readLine(br)
		if line != "" {
			lines = append(lines, line)
			sample.WriteString(line)
			sample.WriteByte('\n')
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return Detection{}, 0, wrap("sample read", err)
		}
	}
	if len(lines) == 0 {
		return Detection{}, 0, fmt.Errorf("converter: empty input %s", path)
	}

	delim := c.cfg.ForceDelimiter
	var delimName DelimiterName
	if delim == 0 {
		delim, delimName = detectDelimiter(sample.String())
	} else {
		delimName = delimiterName(delim)
	}

	first := parseCSVLine(lines[0], delim)
	hasHeader := looksLikeHeader(first)
	var headers []string
	if hasHeader {
		headers = first
	} else {
		headers = syntheticHeaders(len(first))
	}
	mapping := buildMapping(headers)
	// If no ID column detected from header, assume first column is ID for headerless dumps.
	if mapping.IDIndex < 0 {
		mapping.IDIndex = 0
	}

	return Detection{
		Encoding:      enc,
		Delimiter:     delim,
		DelimiterName: delimName,
		HasHeader:     hasHeader,
		Mapping:       mapping,
	}, len(lines), nil
}

func syntheticHeaders(n int) []string {
	// Headerless files: col0=id, remaining extras unless we can guess later.
	out := make([]string, n)
	for i := 0; i < n; i++ {
		if i == 0 {
			out[i] = "id"
			continue
		}
		out[i] = fmt.Sprintf("col_%d", i)
	}
	return out
}

func standardOutputPath(input string) string {
	ext := filepath.Ext(input)
	base := strings.TrimSuffix(input, ext)
	return base + ".standard.csv"
}

func peekFile(path string, n int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, wrap("peek open", err)
	}
	defer f.Close()
	buf := make([]byte, n)
	m, err := io.ReadFull(f, buf)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return buf[:m], nil
	}
	if err != nil {
		return nil, wrap("peek read", err)
	}
	return buf[:m], nil
}

func readLine(r *bufio.Reader) (string, error) {
	var b strings.Builder
	for {
		part, err := r.ReadString('\n')
		b.WriteString(part)
		if err == nil {
			s := b.String()
			s = strings.TrimSuffix(s, "\n")
			s = strings.TrimSuffix(s, "\r")
			return s, nil
		}
		if err == io.EOF {
			s := b.String()
			s = strings.TrimSuffix(s, "\n")
			s = strings.TrimSuffix(s, "\r")
			if s == "" {
				return "", io.EOF
			}
			return s, io.EOF
		}
		return "", err
	}
}

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

// FormatDryRun returns a human-readable dry-run report.
func FormatDryRun(r DryRunReport) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("File: %s\n", r.File))
	b.WriteString(fmt.Sprintf("Encoding: %s\n", r.Detection.Encoding))
	b.WriteString(fmt.Sprintf("Delimiter: %s (%q)\n", r.Detection.DelimiterName, string(r.Detection.Delimiter)))
	b.WriteString(fmt.Sprintf("Has Header: %v\n", r.Detection.HasHeader))
	b.WriteString(fmt.Sprintf("Sample Rows: %d\n", r.SampleRows))
	m := r.Detection.Mapping
	b.WriteString("Detected Mapping:\n")
	b.WriteString(fmt.Sprintf("  id       -> %s\n", headerLabel(m.Headers, m.IDIndex)))
	b.WriteString(fmt.Sprintf("  name     -> %s\n", headerLabel(m.Headers, m.NameIndex)))
	b.WriteString(fmt.Sprintf("  lastname -> %s\n", headerLabel(m.Headers, m.LastNameIndex)))
	b.WriteString(fmt.Sprintf("  phone    -> %s\n", headerLabel(m.Headers, m.PhoneIndex)))
	b.WriteString(fmt.Sprintf("  username -> %s\n", headerLabel(m.Headers, m.UsernameIndex)))
	b.WriteString("Detected Extras:\n")
	if len(r.ExtrasKeys) == 0 {
		b.WriteString("  (none)\n")
	} else {
		for _, k := range r.ExtrasKeys {
			b.WriteString("  - " + k + "\n")
		}
	}
	return b.String()
}

func headerLabel(headers []string, idx int) string {
	if idx < 0 || idx >= len(headers) {
		return "(not found)"
	}
	return fmt.Sprintf("[%d] %s", idx, headers[idx])
}
