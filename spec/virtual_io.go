package spec

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bancek/go-sass/sassio"
)

// VirtualIO implements sassio.IO backed by an in-memory filesystem.
type VirtualIO struct {
	files    map[string]string // absolute path → content
	output   bytes.Buffer
	exitCode int
	fallback sassio.IO // if set, delegates not-found operations to this IO
}

// SetFallback configures a fallback IO to consult when a file is not found
// in the in-memory filesystem. This allows HRX tests to access real shared
// files (e.g. _utils.scss) that exist alongside the HRX archives.
func (v *VirtualIO) SetFallback(io sassio.IO) { v.fallback = io }

// NewVirtualIO creates a VirtualIO populated with the given flat file map.
// The keys should be absolute OS paths.
func NewVirtualIO(files map[string]string) *VirtualIO {
	if files == nil {
		files = make(map[string]string)
	}
	return &VirtualIO{
		files: files,
	}
}

// OutputBuffer returns the captured PrintOutput content.
func (v *VirtualIO) OutputBuffer() string {
	return v.output.String()
}

func (v *VirtualIO) ReadFile(name string) ([]byte, error) {
	name = filepath.Clean(name)
	if content, ok := v.files[name]; ok {
		return []byte(content), nil
	}
	if v.fallback != nil {
		return v.fallback.ReadFile(name)
	}
	return nil, &sassio.FileSystemException{
		Message: "file not found",
		Path:    name,
	}
}

func (v *VirtualIO) WriteFile(path string, contents []byte) error {
	v.files[filepath.Clean(path)] = string(contents)
	return nil
}

func (v *VirtualIO) DeleteFile(path string) error {
	delete(v.files, filepath.Clean(path))
	return nil
}

func (v *VirtualIO) ReadStdin() ([]byte, error) {
	return nil, nil
}

func (v *VirtualIO) FileExists(path string) bool {
	path = filepath.Clean(path)
	if _, ok := v.files[path]; ok {
		return true
	}
	if v.fallback != nil {
		return v.fallback.FileExists(path)
	}
	return false
}

func (v *VirtualIO) DirExists(path string) bool {
	path = filepath.Clean(path)
	if !strings.HasSuffix(path, string(filepath.Separator)) {
		path += string(filepath.Separator)
	}
	for p := range v.files {
		if strings.HasPrefix(filepath.Clean(p), path) {
			return true
		}
	}
	if v.fallback != nil {
		return v.fallback.DirExists(path)
	}
	return false
}

func (v *VirtualIO) LinkExists(path string) bool {
	return false
}

func (v *VirtualIO) EnsureDir(path string) error {
	return nil
}

func (v *VirtualIO) ListDir(path string, recursive bool) ([]string, error) {
	path = filepath.Clean(path)
	if !strings.HasSuffix(path, string(filepath.Separator)) {
		path += string(filepath.Separator)
	}
	var result []string
	seen := make(map[string]bool)
	for p := range v.files {
		p = filepath.Clean(p)
		if !strings.HasPrefix(p, path) {
			continue
		}
		rel := p[len(path):]
		if !recursive && strings.Contains(rel, string(filepath.Separator)) {
			continue
		}
		if !seen[p] {
			result = append(result, p)
			seen[p] = true
		}
	}
	return result, nil
}

func (v *VirtualIO) Realpath(path string) (string, error) {
	return filepath.Clean(path), nil
}

func (v *VirtualIO) ModificationTime(path string) (time.Time, error) {
	if _, ok := v.files[filepath.Clean(path)]; ok {
		return time.Time{}, nil
	}
	if v.fallback != nil {
		return v.fallback.ModificationTime(path)
	}
	return time.Time{}, &sassio.FileSystemException{
		Message: "file not found",
		Path:    path,
	}
}

func (v *VirtualIO) GetEnvironmentVariable(name string) string {
	return ""
}

func (v *VirtualIO) ExitCode() int {
	return v.exitCode
}

func (v *VirtualIO) SetExitCode(code int) {
	v.exitCode = code
}

func (v *VirtualIO) Canonicalize(path string) (string, error) {
	return filepath.Clean(path), nil
}

func (v *VirtualIO) PrintOutput(message string) {
	v.output.WriteString(message)
}

func (v *VirtualIO) SafePrint(message string) {}

func (v *VirtualIO) PrintError(message string) {}

func (v *VirtualIO) IsWindows() bool {
	return false
}

func (v *VirtualIO) IsMacOS() bool {
	return false
}

func (v *VirtualIO) HasTerminal() bool {
	return false
}

func (v *VirtualIO) SupportsAnsiEscapes() bool {
	return false
}

func (v *VirtualIO) Stat(name string) (os.FileInfo, error) {
	name = filepath.Clean(name)
	if content, ok := v.files[name]; ok {
		return &virtualFileInfo{name: filepath.Base(name), size: int64(len(content))}, nil
	}
	// Check if it's a directory (any file has this dir as prefix).
	prefix := name
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	for p := range v.files {
		if strings.HasPrefix(filepath.Clean(p), prefix) {
			return &virtualFileInfo{name: filepath.Base(name), size: 0, isDir: true}, nil
		}
	}
	if v.fallback != nil {
		return v.fallback.Stat(name)
	}
	return nil, &sassio.FileSystemException{
		Message: "no such file or directory",
		Path:    name,
	}
}

type virtualFileInfo struct {
	name  string
	size  int64
	isDir bool
}

func (f *virtualFileInfo) Name() string       { return f.name }
func (f *virtualFileInfo) Size() int64        { return f.size }
func (f *virtualFileInfo) Mode() os.FileMode  { return 0644 }
func (f *virtualFileInfo) ModTime() time.Time { return time.Time{} }
func (f *virtualFileInfo) IsDir() bool        { return f.isDir }
func (f *virtualFileInfo) Sys() any           { return nil }

var _ fs.FileInfo = (*virtualFileInfo)(nil)
