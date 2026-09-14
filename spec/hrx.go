package spec

import (
	"bufio"
	"fmt"
	"path"
	"sort"
	"strings"
)

// HrxArchive is a parsed HRX file representing a virtual directory tree.
type HrxArchive struct {
	// Path is the logical path of this directory (e.g. "core_functions/color/adjust/rgb").
	Path string
	// Files maps basename to content.
	Files map[string]string
	// Subdirs maps basename to subdirectory.
	Subdirs map[string]*HrxArchive
}

// ParseHRX parses an HRX archive string into a directory tree.
func ParseHRX(archivePath string, content string) (*HrxArchive, error) {
	type entry struct {
		relPath string
		body    strings.Builder
	}
	var entries []entry
	var current *entry

	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "<===>") {
			if current != nil {
				s := current.body.String()
				strings.TrimSuffix(s, "\n")
				entries = append(entries, entry{relPath: current.relPath, body: strings.Builder{}})
				entries[len(entries)-1].body.WriteString(s)
			}
			rest := strings.TrimPrefix(line, "<===>")
			rest = strings.TrimLeft(rest, " \t")
			if rest == "" || strings.HasPrefix(rest, "===") {
				current = nil
				continue
			}
			current = &entry{relPath: rest}
		} else if current != nil {
			current.body.WriteString(line)
			current.body.WriteString("\n")
		}
	}
	if current != nil {
		s := current.body.String()
		strings.TrimSuffix(s, "\n")
		entries = append(entries, entry{relPath: current.relPath, body: strings.Builder{}})
		entries[len(entries)-1].body.WriteString(s)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("parsing HRX %s: %w", archivePath, err)
	}

	archive := &HrxArchive{
		Path:    "",
		Files:   make(map[string]string),
		Subdirs: make(map[string]*HrxArchive),
	}
	for _, e := range entries {
		insertEntry(archive, e.relPath, e.body.String())
	}
	return archive, nil
}

func insertEntry(root *HrxArchive, relPath, body string) {
	parts := splitPath(relPath)
	if len(parts) == 0 {
		return
	}
	current := root
	for i, p := range parts {
		if i == len(parts)-1 {
			current.Files[p] = body
		} else {
			if current.Subdirs[p] == nil {
				subPath := path.Join(current.Path, p)
				current.Subdirs[p] = &HrxArchive{
					Path:    subPath,
					Files:   make(map[string]string),
					Subdirs: make(map[string]*HrxArchive),
				}
			}
			current = current.Subdirs[p]
		}
	}
}

func splitPath(p string) []string {
	p = path.Clean(p)
	if p == "." || p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

// IsTestDir reports whether this archive or directory contains a Sass input file.
func (a *HrxArchive) IsTestDir() bool {
	return a != nil && (a.hasFile("input.scss") || a.hasFile("input.sass"))
}

func (a *HrxArchive) hasFile(name string) bool {
	_, ok := a.Files[name]
	return ok
}

// InputFile returns the name of the input file ("input.scss" or "input.sass").
func (a *HrxArchive) InputFile() string {
	if _, ok := a.Files["input.scss"]; ok {
		return "input.scss"
	}
	return "input.sass"
}

// AllFiles returns the content of all files, including those in subdirectories,
// keyed by their relative path from this archive root.
func (a *HrxArchive) AllFiles() map[string]string {
	result := make(map[string]string)
	for name, content := range a.Files {
		result[name] = content
	}
	for name, sub := range a.Subdirs {
		for p, content := range sub.AllFiles() {
			result[path.Join(name, p)] = content
		}
	}
	return result
}

// ListTestDirs returns all sub-archives that are test directories (have input.scss/sass),
// including this one if it is a test dir.
func (a *HrxArchive) ListTestDirs() []*HrxArchive {
	var result []*HrxArchive
	if a.IsTestDir() {
		result = append(result, a)
	}
	// Process subdirs in sorted order for deterministic output.
	names := make([]string, 0, len(a.Subdirs))
	for name := range a.Subdirs {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		result = append(result, a.Subdirs[name].ListTestDirs()...)
	}
	return result
}

// GetOptionsYAML returns the content of options.yml if present.
func (a *HrxArchive) GetOptionsYAML() string {
	if content, ok := a.Files["options.yml"]; ok {
		return content
	}
	return ""
}
