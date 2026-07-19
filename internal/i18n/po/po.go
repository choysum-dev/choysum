// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package po

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
)

// Entry is one gettext message (supports msgctxt; MVP singular only).
type Entry struct {
	TranslatorComments []string
	ExtractedComments  []string
	References         []string
	Flags              []string
	Msgctxt            string
	Msgid              string
	Msgstr             string
	Obsolete           bool
}

// Key returns the sync/import lookup key (msgctxt, msgid, kind).
// Kind comes from `#. kind:` extracted comments; missing kind means "literal".
func (e Entry) Key() string {
	return e.Msgctxt + "\x00" + e.Msgid + "\x00" + e.Kind()
}

// Kind returns the terminology kind from `#. kind:` comments (default "literal").
func (e Entry) Kind() string {
	for _, c := range e.ExtractedComments {
		c = strings.TrimSpace(c)
		lower := strings.ToLower(c)
		if strings.HasPrefix(lower, "kind:") {
			kind := strings.TrimSpace(c[len("kind:"):])
			if kind == "" {
				return "literal"
			}
			return kind
		}
	}
	return "literal"
}

// Parse reads a PO/POT file into entries (including obsolete #~ blocks).
func Parse(r io.Reader) ([]Entry, error) {
	scanner := bufio.NewScanner(r)
	// Allow long msgstr lines.
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var entries []Entry
	var cur Entry
	var hasEntry bool
	state := ""

	flush := func() {
		if !hasEntry {
			return
		}
		// Skip header msgid "" with empty msgctxt at file start is still an entry;
		// callers may filter it.
		entries = append(entries, cur)
		cur = Entry{}
		hasEntry = false
		state = ""
	}

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		isObsoleteLine := strings.HasPrefix(trimmed, "#~")
		if isObsoleteLine {
			trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "#~"))
		}

		if trimmed == "" {
			flush()
			continue
		}

		if strings.HasPrefix(trimmed, "#") {
			// Comments belong to the following entry. If the current entry is
			// already complete (msgstr seen) and the file omits a blank line,
			// flush before attaching the comment to the next entry.
			if hasEntry && cur.Msgid != "" && state == "msgstr" {
				flush()
			}
			parseComment(&cur, trimmed)
			hasEntry = true
			if isObsoleteLine {
				cur.Obsolete = true
			}
			continue
		}

		if isObsoleteLine {
			cur.Obsolete = true
		}

		switch {
		case strings.HasPrefix(trimmed, "msgctxt"):
			if hasEntry && cur.Msgid != "" {
				flush()
				if isObsoleteLine {
					cur.Obsolete = true
				}
			}
			cur.Msgctxt = parsePoKeyword(trimmed, "msgctxt")
			hasEntry = true
			state = "msgctxt"
		case strings.HasPrefix(trimmed, "msgid_plural"):
			// MVP: ignore plural body beyond storing nothing special.
			state = "msgid_plural"
			hasEntry = true
		case strings.HasPrefix(trimmed, "msgid"):
			if hasEntry && cur.Msgid != "" && state != "msgctxt" {
				flush()
				if isObsoleteLine {
					cur.Obsolete = true
				}
			}
			cur.Msgid = parsePoKeyword(trimmed, "msgid")
			hasEntry = true
			state = "msgid"
		case strings.HasPrefix(trimmed, "msgstr"):
			// msgstr or msgstr[n] — tolerate tabs/extra spaces after the keyword.
			rest := strings.TrimPrefix(trimmed, "msgstr")
			if strings.HasPrefix(rest, "[") {
				if idx := strings.Index(rest, "]"); idx >= 0 {
					rest = rest[idx+1:]
				}
			}
			cur.Msgstr = unquotePo(rest)
			hasEntry = true
			state = "msgstr"
		case strings.HasPrefix(trimmed, "\""):
			chunk := unquotePo(trimmed)
			switch state {
			case "msgctxt":
				cur.Msgctxt += chunk
			case "msgid":
				cur.Msgid += chunk
			case "msgstr":
				cur.Msgstr += chunk
			}
		default:
			return nil, fmt.Errorf("unsupported po line: %q", line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	flush()
	return entries, nil
}

func parseComment(cur *Entry, trimmed string) {
	switch {
	case strings.HasPrefix(trimmed, "#:"):
		refs := strings.Fields(strings.TrimSpace(strings.TrimPrefix(trimmed, "#:")))
		cur.References = append(cur.References, refs...)
	case strings.HasPrefix(trimmed, "#."):
		cur.ExtractedComments = append(cur.ExtractedComments, strings.TrimSpace(strings.TrimPrefix(trimmed, "#.")))
	case strings.HasPrefix(trimmed, "#,"):
		flags := strings.Split(strings.TrimSpace(strings.TrimPrefix(trimmed, "#,")), ",")
		for _, f := range flags {
			f = strings.TrimSpace(f)
			if f != "" {
				cur.Flags = append(cur.Flags, f)
			}
		}
	case strings.HasPrefix(trimmed, "#"):
		cur.TranslatorComments = append(cur.TranslatorComments, strings.TrimSpace(strings.TrimPrefix(trimmed, "#")))
	}
}

func parsePoKeyword(line, keyword string) string {
	rest := strings.TrimSpace(strings.TrimPrefix(line, keyword))
	return unquotePo(rest)
}

func unquotePo(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 2 {
		return s
	}
	v, err := strconv.Unquote(s)
	if err != nil {
		// Fallback: strip surrounding quotes naively.
		if s[0] == '"' && s[len(s)-1] == '"' {
			return s[1 : len(s)-1]
		}
		return s
	}
	return v
}

// Write writes entries as a GNU gettext PO file.
func Write(w io.Writer, entries []Entry) error {
	var buf bytes.Buffer
	for i, e := range entries {
		if i > 0 {
			buf.WriteByte('\n')
		}
		prefix := ""
		if e.Obsolete {
			prefix = "#~ "
		}
		for _, c := range e.TranslatorComments {
			if e.Obsolete {
				buf.WriteString("#~ # ")
			} else {
				buf.WriteString("# ")
			}
			buf.WriteString(c)
			buf.WriteByte('\n')
		}
		for _, c := range e.ExtractedComments {
			if e.Obsolete {
				buf.WriteString("#~ #. ")
			} else {
				buf.WriteString("#. ")
			}
			buf.WriteString(c)
			buf.WriteByte('\n')
		}
		if len(e.References) > 0 && !e.Obsolete {
			buf.WriteString("#: ")
			buf.WriteString(strings.Join(e.References, " "))
			buf.WriteByte('\n')
		}
		if len(e.Flags) > 0 && !e.Obsolete {
			buf.WriteString("#, ")
			buf.WriteString(strings.Join(e.Flags, ", "))
			buf.WriteByte('\n')
		}
		if e.Msgctxt != "" {
			buf.WriteString(prefix)
			buf.WriteString("msgctxt ")
			buf.WriteString(strconv.Quote(e.Msgctxt))
			buf.WriteByte('\n')
		}
		buf.WriteString(prefix)
		buf.WriteString("msgid ")
		buf.WriteString(strconv.Quote(e.Msgid))
		buf.WriteByte('\n')
		buf.WriteString(prefix)
		buf.WriteString("msgstr ")
		buf.WriteString(strconv.Quote(e.Msgstr))
		buf.WriteByte('\n')
	}
	_, err := io.Copy(w, &buf)
	return err
}

// SortEntries orders by obsolete last, then msgctxt, msgid, kind.
func SortEntries(entries []Entry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Obsolete != entries[j].Obsolete {
			return !entries[i].Obsolete && entries[j].Obsolete
		}
		if entries[i].Msgctxt != entries[j].Msgctxt {
			return entries[i].Msgctxt < entries[j].Msgctxt
		}
		if entries[i].Msgid != entries[j].Msgid {
			return entries[i].Msgid < entries[j].Msgid
		}
		return entries[i].Kind() < entries[j].Kind()
	})
}

// IsHeader reports the empty msgid header entry.
func IsHeader(e Entry) bool {
	return e.Msgid == "" && !e.Obsolete
}
