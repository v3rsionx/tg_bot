package converter

import (
	"bytes"
	"encoding/binary"
	"io"
	"unicode/utf16"
	"unicode/utf8"
)

// detectEncoding sniffs BOM / heuristics from a prefix of the file.
func detectEncoding(prefix []byte) EncodingName {
	if len(prefix) >= 3 && prefix[0] == 0xEF && prefix[1] == 0xBB && prefix[2] == 0xBF {
		return EncodingUTF8BOM
	}
	if len(prefix) >= 2 && prefix[0] == 0xFF && prefix[1] == 0xFE {
		return EncodingUTF16LE
	}
	if len(prefix) >= 2 && prefix[0] == 0xFE && prefix[1] == 0xFF {
		return EncodingUTF16BE
	}
	if looksLikeUTF16LE(prefix) {
		return EncodingUTF16LE
	}
	if looksLikeUTF16BE(prefix) {
		return EncodingUTF16BE
	}
	if utf8.Valid(prefix) {
		return EncodingUTF8
	}
	if looksLikeCP1251(prefix) {
		return EncodingWindows1251
	}
	return EncodingLatin1
}

func looksLikeUTF16LE(b []byte) bool {
	if len(b) < 8 {
		return false
	}
	zeros := 0
	for i := 1; i < len(b) && i < 64; i += 2 {
		if b[i] == 0 {
			zeros++
		}
	}
	return zeros >= 8
}

func looksLikeUTF16BE(b []byte) bool {
	if len(b) < 8 {
		return false
	}
	zeros := 0
	for i := 0; i < len(b) && i < 64; i += 2 {
		if b[i] == 0 {
			zeros++
		}
	}
	return zeros >= 8
}

func looksLikeCP1251(b []byte) bool {
	// Heuristic: high bytes in Cyrillic windows-1251 ranges.
	cyr := 0
	for _, v := range b {
		if (v >= 0xC0 && v <= 0xFF) || v == 0xA8 || v == 0xB8 {
			cyr++
		}
	}
	return cyr > len(b)/10
}

// newDecodingReader wraps r to produce UTF-8 text for the detected encoding.
func newDecodingReader(r io.Reader, enc EncodingName) (io.Reader, error) {
	switch enc {
	case EncodingUTF8:
		return r, nil
	case EncodingUTF8BOM:
		return newBOMSkipReader(r, []byte{0xEF, 0xBB, 0xBF}), nil
	case EncodingUTF16LE:
		return newUTF16Reader(r, binary.LittleEndian, true), nil
	case EncodingUTF16BE:
		return newUTF16Reader(r, binary.BigEndian, true), nil
	case EncodingWindows1251:
		return &byteMapReader{r: r, table: cp1251Table}, nil
	case EncodingLatin1:
		return &byteMapReader{r: r, table: latin1Table}, nil
	default:
		return r, nil
	}
}

type bomSkipReader struct {
	r      io.Reader
	bom    []byte
	checked bool
	prefix []byte
}

func newBOMSkipReader(r io.Reader, bom []byte) *bomSkipReader {
	return &bomSkipReader{r: r, bom: bom}
}

func (b *bomSkipReader) Read(p []byte) (int, error) {
	if !b.checked {
		b.checked = true
		buf := make([]byte, len(b.bom))
		n, err := io.ReadFull(b.r, buf)
		if n > 0 && bytes.Equal(buf[:n], b.bom[:n]) && n == len(b.bom) {
			// BOM consumed.
		} else if n > 0 {
			b.prefix = buf[:n]
		}
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return 0, err
		}
	}
	if len(b.prefix) > 0 {
		n := copy(p, b.prefix)
		b.prefix = b.prefix[n:]
		return n, nil
	}
	return b.r.Read(p)
}

type utf16Reader struct {
	r       io.Reader
	order   binary.ByteOrder
	skipBOM bool
	buf     []byte
	carry   []byte
}

func newUTF16Reader(r io.Reader, order binary.ByteOrder, skipBOM bool) *utf16Reader {
	return &utf16Reader{r: r, order: order, skipBOM: skipBOM}
}

func (u *utf16Reader) Read(p []byte) (int, error) {
	if len(u.buf) > 0 {
		n := copy(p, u.buf)
		u.buf = u.buf[n:]
		return n, nil
	}
	raw := make([]byte, 4096)
	if len(u.carry) > 0 {
		copy(raw, u.carry)
	}
	n, err := u.r.Read(raw[len(u.carry):])
	n += len(u.carry)
	u.carry = nil
	if n == 0 {
		return 0, err
	}
	if n%2 == 1 {
		u.carry = []byte{raw[n-1]}
		n--
	}
	if n == 0 {
		if err != nil {
			return 0, err
		}
		return 0, nil
	}
	start := 0
	if u.skipBOM && n >= 2 {
		v := u.order.Uint16(raw[:2])
		if v == 0xFEFF {
			start = 2
		}
		u.skipBOM = false
	}
	units := make([]uint16, 0, (n-start)/2)
	for i := start; i+1 < n; i += 2 {
		units = append(units, u.order.Uint16(raw[i:]))
	}
	runes := utf16.Decode(units)
	encoded := make([]byte, 0, len(runes)*3)
	tmp := make([]byte, utf8.UTFMax)
	for _, r := range runes {
		m := utf8.EncodeRune(tmp, r)
		encoded = append(encoded, tmp[:m]...)
	}
	nOut := copy(p, encoded)
	if nOut < len(encoded) {
		u.buf = encoded[nOut:]
	}
	if nOut == 0 && err != nil {
		return 0, err
	}
	return nOut, err
}

type byteMapReader struct {
	r     io.Reader
	table *[256]rune
	buf   []byte
}

func (b *byteMapReader) Read(p []byte) (int, error) {
	if len(b.buf) > 0 {
		n := copy(p, b.buf)
		b.buf = b.buf[n:]
		return n, nil
	}
	raw := make([]byte, 2048)
	n, err := b.r.Read(raw)
	if n == 0 {
		return 0, err
	}
	encoded := make([]byte, 0, n*2)
	tmp := make([]byte, utf8.UTFMax)
	for i := 0; i < n; i++ {
		r := b.table[raw[i]]
		m := utf8.EncodeRune(tmp, r)
		encoded = append(encoded, tmp[:m]...)
	}
	out := copy(p, encoded)
	if out < len(encoded) {
		b.buf = encoded[out:]
	}
	return out, err
}

// latin1Table maps ISO-8859-1 bytes to runes.
var latin1Table = func() *[256]rune {
	var t [256]rune
	for i := 0; i < 256; i++ {
		t[i] = rune(i)
	}
	return &t
}()

// cp1251Table maps Windows-1251 bytes to Unicode runes.
var cp1251Table = func() *[256]rune {
	var t [256]rune
	for i := 0; i < 256; i++ {
		t[i] = rune(i)
	}
	// C1 controls / specials
	t[0x80] = 0x0402
	t[0x81] = 0x0403
	t[0x82] = 0x201A
	t[0x83] = 0x0453
	t[0x84] = 0x201E
	t[0x85] = 0x2026
	t[0x86] = 0x2020
	t[0x87] = 0x2021
	t[0x88] = 0x20AC
	t[0x89] = 0x2030
	t[0x8A] = 0x0409
	t[0x8B] = 0x2039
	t[0x8C] = 0x040A
	t[0x8D] = 0x040C
	t[0x8E] = 0x040B
	t[0x8F] = 0x040F
	t[0x90] = 0x0452
	t[0x91] = 0x2018
	t[0x92] = 0x2019
	t[0x93] = 0x201C
	t[0x94] = 0x201D
	t[0x95] = 0x2022
	t[0x96] = 0x2013
	t[0x97] = 0x2014
	t[0x99] = 0x2122
	t[0x9A] = 0x0459
	t[0x9B] = 0x203A
	t[0x9C] = 0x045A
	t[0x9D] = 0x045C
	t[0x9E] = 0x045B
	t[0x9F] = 0x045F
	t[0xA0] = 0x00A0
	t[0xA1] = 0x040E
	t[0xA2] = 0x045E
	t[0xA3] = 0x0408
	t[0xA4] = 0x00A4
	t[0xA5] = 0x0490
	t[0xA6] = 0x00A6
	t[0xA7] = 0x00A7
	t[0xA8] = 0x0401
	t[0xA9] = 0x00A9
	t[0xAA] = 0x0404
	t[0xAB] = 0x00AB
	t[0xAC] = 0x00AC
	t[0xAD] = 0x00AD
	t[0xAE] = 0x00AE
	t[0xAF] = 0x0407
	t[0xB0] = 0x00B0
	t[0xB1] = 0x00B1
	t[0xB2] = 0x0406
	t[0xB3] = 0x0456
	t[0xB4] = 0x0491
	t[0xB5] = 0x00B5
	t[0xB6] = 0x00B6
	t[0xB7] = 0x00B7
	t[0xB8] = 0x0451
	t[0xB9] = 0x2116
	t[0xBA] = 0x0454
	t[0xBB] = 0x00BB
	t[0xBC] = 0x0458
	t[0xBD] = 0x0405
	t[0xBE] = 0x0455
	t[0xBF] = 0x0457
	for i := 0; i < 64; i++ {
		t[0xC0+i] = rune(0x0410 + i) // А-я
	}
	return &t
}()
