package importer

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
)

// lineReader streams source lines with accurate byte offsets for resume.
type lineReader struct {
	file   *os.File
	reader *bufio.Reader
	offset int64
}

// openLineReader opens path and optionally seeks to resumeOffset.
func openLineReader(path string, resumeOffset int64, bufferSize int) (*lineReader, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("importer: open %q: %w", path, err)
	}

	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, 0, fmt.Errorf("importer: stat %q: %w", path, err)
	}
	total := info.Size()

	if resumeOffset > 0 {
		if resumeOffset > total {
			_ = file.Close()
			return nil, 0, fmt.Errorf("importer: resume offset %d exceeds file size %d for %q", resumeOffset, total, path)
		}
		if _, err := file.Seek(resumeOffset, io.SeekStart); err != nil {
			_ = file.Close()
			return nil, 0, fmt.Errorf("importer: seek %q: %w", path, err)
		}
	}

	return &lineReader{
		file:   file,
		reader: bufio.NewReaderSize(file, bufferSize),
		offset: resumeOffset,
	}, total, nil
}

// Close closes the underlying file.
func (l *lineReader) Close() error {
	if l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	return err
}

// Offset returns the absolute byte offset of the next unread byte.
func (l *lineReader) Offset() int64 {
	return l.offset
}

// ReadLine reads the next logical line and returns the offset after it.
func (l *lineReader) ReadLine(maxLineBytes int) (text string, nextOffset int64, err error) {
	var chunk []byte
	for {
		part, readErr := l.reader.ReadSlice('\n')
		if len(part) > 0 {
			chunk = append(chunk, part...)
			l.offset += int64(len(part))
		}
		if readErr == nil {
			break
		}
		if readErr == bufio.ErrBufferFull {
			if maxLineBytes > 0 && len(chunk) > maxLineBytes {
				return "", l.offset, fmt.Errorf("%w: line exceeds %d bytes", ErrMalformedLine, maxLineBytes)
			}
			continue
		}
		if readErr == io.EOF {
			if len(chunk) == 0 {
				return "", l.offset, io.EOF
			}
			break
		}
		return "", l.offset, readErr
	}

	if maxLineBytes > 0 && len(chunk) > maxLineBytes {
		return "", l.offset, fmt.Errorf("%w: line exceeds %d bytes", ErrMalformedLine, maxLineBytes)
	}

	chunk = bytes.TrimRight(chunk, "\r\n")
	return string(chunk), l.offset, nil
}

// parseFields parses one CSV/TXT line using delimiter and quoted-field rules.
func parseFields(line string, delimiter rune) ([]string, error) {
	reader := csv.NewReader(strings.NewReader(line))
	reader.Comma = delimiter
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true
	reader.ReuseRecord = false

	fields, err := reader.Read()
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("%w: empty line", ErrMalformedLine)
		}
		return nil, fmt.Errorf("%w: %v", ErrMalformedLine, err)
	}
	return fields, nil
}
