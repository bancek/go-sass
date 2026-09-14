package sassio

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func newDefaultIO() *DefaultIO {
	return NewDefaultIO()
}

func TestDefaultIO_ReadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	d := newDefaultIO()
	data, err := d.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("got %q, want %q", string(data), "hello")
	}
}

func TestDefaultIO_ReadFile_NotFound(t *testing.T) {
	d := newDefaultIO()
	_, err := d.ReadFile("/nonexistent-file-for-test")
	if err == nil {
		t.Fatal("expected error")
	}
	fse, ok := errors.AsType[*FileSystemException](err)
	if !ok {
		t.Fatalf("error type %T, want *FileSystemException", err)
	}
	if fse.Path != "/nonexistent-file-for-test" {
		t.Fatalf("path %q, want %q", fse.Path, "/nonexistent-file-for-test")
	}
	if fse.Err == nil {
		t.Fatal("expected wrapped error")
	}
	if !fse.IsNotExist() {
		t.Fatal("expected IsNotExist() to be true")
	}
}

func TestDefaultIO_Stat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stat.txt")
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	d := newDefaultIO()
	info, err := d.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.IsDir() {
		t.Fatal("expected regular file")
	}
}

func TestDefaultIO_Stat_NotFound(t *testing.T) {
	d := newDefaultIO()
	_, err := d.Stat("/nonexistent-file-for-test")
	if err == nil {
		t.Fatal("expected error")
	}
	fse, ok := errors.AsType[*FileSystemException](err)
	if !ok {
		t.Fatalf("error type %T, want *FileSystemException", err)
	}
	if fse.Path != "/nonexistent-file-for-test" {
		t.Fatalf("path %q, want %q", fse.Path, "/nonexistent-file-for-test")
	}
	if !fse.IsNotExist() {
		t.Fatal("expected IsNotExist() to be true")
	}
}

func TestDefaultIO_Canonicalize_Absolute(t *testing.T) {
	d := newDefaultIO()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.scss")
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	result, err := d.Canonicalize(path)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(result) {
		t.Fatalf("expected absolute path, got %q", result)
	}
}

func TestDefaultIO_Canonicalize_Symlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "alink")
	if err := os.WriteFile(target, nil, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlinks not supported:", err)
	}

	d := newDefaultIO()
	result, err := d.Canonicalize(link)
	if err != nil {
		t.Fatal(err)
	}
	// Canonicalize is lexical — it preserves symlink names (matching Dart).
	// On case-insensitive systems realCasePath is applied, but the temp dir's
	// case is already correct.
	expected, err := filepath.Abs(link)
	if err != nil {
		t.Fatal(err)
	}
	if result != expected {
		t.Fatalf("got %q, want %q", result, expected)
	}
}

func TestDefaultIO_Canonicalize_Case(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("case-correction test only meaningful on macOS")
	}

	d := newDefaultIO()
	dir := t.TempDir()

	realName := "MixedCase_SCSS.scss"
	realPath := filepath.Join(dir, realName)
	if err := os.WriteFile(realPath, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	wrongCase := filepath.Join(dir, "mixedcase_scss.scss")
	result, err := d.Canonicalize(wrongCase)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasSuffix(result, realName) {
		t.Fatalf("expected suffix %q, got %q", realName, result)
	}
}

func TestDefaultIO_Canonicalize_Nonexistent(t *testing.T) {
	d := newDefaultIO()
	result, err := d.Canonicalize("/nonexistent-path-for-test/foo.scss")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(result) {
		t.Fatalf("expected absolute path, got %q", result)
	}
}

func TestDefaultIO_WriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "write.txt")

	d := newDefaultIO()
	if err := d.WriteFile(path, []byte("hello")); err != nil {
		t.Fatal(err)
	}

	data, err := d.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("got %q, want %q", string(data), "hello")
	}
}

func TestDefaultIO_WriteFile_PermissionDenied(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nope.txt")
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	// Read-only on the FILE itself: a read-only directory still accepts
	// new files on Windows (the attribute is advisory there), so
	// dir-level readonly is a unix-only construction.
	if err := os.Chmod(path, 0444); err != nil {
		t.Fatal(err)
	}
	// Restore first: tempdir cleanup cannot delete read-only files on
	// Windows.
	defer os.Chmod(path, 0644)

	d := newDefaultIO()
	err := d.WriteFile(path, []byte("x"))
	if err == nil {
		t.Fatal("expected error")
	}
	fse, ok := errors.AsType[*FileSystemException](err)
	if !ok {
		t.Fatalf("error type %T, want *FileSystemException", err)
	}
	if fse.Path != path {
		t.Fatalf("path %q, want %q", fse.Path, path)
	}
	if !fse.IsPermission() {
		t.Fatal("expected IsPermission() to be true")
	}
}

func TestDefaultIO_DeleteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "del.txt")
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	d := newDefaultIO()
	if err := d.DeleteFile(path); err != nil {
		t.Fatal(err)
	}
	if d.FileExists(path) {
		t.Fatal("expected file to be deleted")
	}
}

func TestDefaultIO_DeleteFile_NotFound(t *testing.T) {
	d := newDefaultIO()
	err := d.DeleteFile("/nonexistent-file-for-test")
	if err == nil {
		t.Fatal("expected error")
	}
	fse, ok := errors.AsType[*FileSystemException](err)
	if !ok {
		t.Fatalf("error type %T, want *FileSystemException", err)
	}
	if fse.Path != "/nonexistent-file-for-test" {
		t.Fatalf("path %q, want %q", fse.Path, "/nonexistent-file-for-test")
	}
	if !fse.IsNotExist() {
		t.Fatal("expected IsNotExist() to be true")
	}
}

func TestDefaultIO_FileExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "exists.txt")
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	d := newDefaultIO()
	if !d.FileExists(path) {
		t.Fatal("expected file to exist")
	}
}

func TestDefaultIO_FileExists_NotFound(t *testing.T) {
	d := newDefaultIO()
	if d.FileExists("/nonexistent-file-for-test") {
		t.Fatal("expected file to not exist")
	}
}

func TestDefaultIO_FileExists_IsDir(t *testing.T) {
	dir := t.TempDir()
	d := newDefaultIO()
	if d.FileExists(dir) {
		t.Fatal("expected directory to return false for FileExists")
	}
}

func TestDefaultIO_DirExists(t *testing.T) {
	dir := t.TempDir()
	d := newDefaultIO()
	if !d.DirExists(dir) {
		t.Fatal("expected dir to exist")
	}
}

func TestDefaultIO_DirExists_NotFound(t *testing.T) {
	d := newDefaultIO()
	if d.DirExists("/nonexistent-dir-for-test") {
		t.Fatal("expected dir to not exist")
	}
}

func TestDefaultIO_DirExists_IsFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	d := newDefaultIO()
	if d.DirExists(path) {
		t.Fatal("expected file to return false for DirExists")
	}
}

func TestDefaultIO_LinkExists(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "link_target")
	link := filepath.Join(dir, "the_link")
	if err := os.WriteFile(target, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skip("symlinks not supported:", err)
	}

	d := newDefaultIO()
	if !d.LinkExists(link) {
		t.Fatal("expected symlink to exist")
	}
}

func TestDefaultIO_LinkExists_NotFound(t *testing.T) {
	d := newDefaultIO()
	if d.LinkExists("/nonexistent-link-for-test") {
		t.Fatal("expected link to not exist")
	}
}

func TestDefaultIO_LinkExists_RegularFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "regular.txt")
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	d := newDefaultIO()
	if d.LinkExists(path) {
		t.Fatal("expected regular file to return false for LinkExists")
	}
}

func TestDefaultIO_EnsureDir(t *testing.T) {
	dir := t.TempDir()
	newDir := filepath.Join(dir, "newdir")

	d := newDefaultIO()
	if err := d.EnsureDir(newDir); err != nil {
		t.Fatal(err)
	}
	if !d.DirExists(newDir) {
		t.Fatal("expected dir to exist after EnsureDir")
	}
}

func TestDefaultIO_EnsureDir_Nested(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "a", "b", "c")

	d := newDefaultIO()
	if err := d.EnsureDir(nested); err != nil {
		t.Fatal(err)
	}
	if !d.DirExists(nested) {
		t.Fatal("expected nested dir to exist after EnsureDir")
	}
}

func TestDefaultIO_ListDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}

	d := newDefaultIO()
	files, err := d.ListDir(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2: %v", len(files), files)
	}
}

func TestDefaultIO_ListDir_Empty(t *testing.T) {
	dir := t.TempDir()
	d := newDefaultIO()
	files, err := d.ListDir(dir, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 0 {
		t.Fatalf("got %d files, want 0: %v", len(files), files)
	}
}

func TestDefaultIO_ListDir_Recursive(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "top.txt"), []byte("t"), 0644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "sub")
	if err := os.Mkdir(sub, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "nested.txt"), []byte("n"), 0644); err != nil {
		t.Fatal(err)
	}

	d := newDefaultIO()
	files, err := d.ListDir(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2: %v", len(files), files)
	}
}

func TestDefaultIO_ListDir_NotFound(t *testing.T) {
	d := newDefaultIO()
	_, err := d.ListDir("/nonexistent-dir-for-test", false)
	if err == nil {
		t.Fatal("expected error")
	}
	fse, ok := errors.AsType[*FileSystemException](err)
	if !ok {
		t.Fatalf("error type %T, want *FileSystemException", err)
	}
	if fse.Path != "/nonexistent-dir-for-test" {
		t.Fatalf("path %q, want %q", fse.Path, "/nonexistent-dir-for-test")
	}
	if !fse.IsNotExist() {
		t.Fatal("expected IsNotExist() to be true")
	}
}

func TestDefaultIO_Realpath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "real.txt")
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	d := newDefaultIO()
	result, err := d.Realpath(path)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(result) {
		t.Fatalf("expected absolute path, got %q", result)
	}
}

func TestDefaultIO_Realpath_NotFound(t *testing.T) {
	d := newDefaultIO()
	_, err := d.Realpath("/nonexistent-file-for-test")
	if err == nil {
		t.Fatal("expected error")
	}
	fse, ok := errors.AsType[*FileSystemException](err)
	if !ok {
		t.Fatalf("error type %T, want *FileSystemException", err)
	}
	if fse.Path != "/nonexistent-file-for-test" {
		t.Fatalf("path %q, want %q", fse.Path, "/nonexistent-file-for-test")
	}
	if !fse.IsNotExist() {
		t.Fatal("expected IsNotExist() to be true")
	}
}

func TestDefaultIO_ModificationTime(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "modtime.txt")
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	d := newDefaultIO()
	mtime, err := d.ModificationTime(path)
	if err != nil {
		t.Fatal(err)
	}
	if mtime.IsZero() {
		t.Fatal("expected non-zero modification time")
	}
}

func TestDefaultIO_ModificationTime_NotFound(t *testing.T) {
	d := newDefaultIO()
	_, err := d.ModificationTime("/nonexistent-file-for-test")
	if err == nil {
		t.Fatal("expected error")
	}
	fse, ok := errors.AsType[*FileSystemException](err)
	if !ok {
		t.Fatalf("error type %T, want *FileSystemException", err)
	}
	if fse.Path != "/nonexistent-file-for-test" {
		t.Fatalf("path %q, want %q", fse.Path, "/nonexistent-file-for-test")
	}
	if !fse.IsNotExist() {
		t.Fatal("expected IsNotExist() to be true")
	}
}

func TestDefaultIO_GetEnvironmentVariable_Missing(t *testing.T) {
	d := newDefaultIO()
	result := d.GetEnvironmentVariable("__NONEXISTENT_ENV_VAR_FOR_TESTS__")
	if result != "" {
		t.Fatalf("got %q, want empty string for missing env var", result)
	}
}

func TestDefaultIO_ExitCode(t *testing.T) {
	d := newDefaultIO()
	if d.ExitCode() != 0 {
		t.Fatalf("default exit code %d, want 0", d.ExitCode())
	}
}

func TestDefaultIO_ExitCode_SetGet(t *testing.T) {
	d := newDefaultIO()
	d.SetExitCode(42)
	if d.ExitCode() != 42 {
		t.Fatalf("exit code %d, want 42", d.ExitCode())
	}
}

func TestDefaultIO_IsWindows(t *testing.T) {
	d := newDefaultIO()
	got := d.IsWindows()
	if runtime.GOOS == "windows" && !got {
		t.Fatal("expected IsWindows to be true on Windows")
	}
	if runtime.GOOS != "windows" && got {
		t.Fatal("expected IsWindows to be false on non-Windows")
	}
}

func TestDefaultIO_IsMacOS(t *testing.T) {
	d := newDefaultIO()
	got := d.IsMacOS()
	if runtime.GOOS == "darwin" && !got {
		t.Fatal("expected IsMacOS to be true on macOS")
	}
	if runtime.GOOS != "darwin" && got {
		t.Fatal("expected IsMacOS to be false on non-macOS")
	}
}

func TestDefaultIO_Canonicalize_Relative(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	path := filepath.Join(dir, "relative.scss")
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	d := newDefaultIO()
	result, err := d.Canonicalize("relative.scss")
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(result) {
		t.Fatalf("expected absolute path from relative input, got %q", result)
	}
}

func TestDefaultIO_Canonicalize_DotDot(t *testing.T) {
	dir := t.TempDir()
	orig, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(orig)

	path := filepath.Join(dir, "style.scss")
	if err := os.WriteFile(path, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	d := newDefaultIO()
	result, err := d.Canonicalize(filepath.Join("..", filepath.Base(dir), "style.scss"))
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(result) {
		t.Fatalf("expected absolute path, got %q", result)
	}
}

func TestDefaultIO_Canonicalize_BrokenSymlink(t *testing.T) {
	dir := t.TempDir()
	broken := filepath.Join(dir, "broken_link")
	if err := os.Symlink("/nonexistent/target", broken); err != nil {
		t.Skip("symlinks not supported:", err)
	}

	d := newDefaultIO()
	result, err := d.Canonicalize(broken)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(result) {
		t.Fatalf("expected absolute path for broken symlink, got %q", result)
	}
}

func TestDefaultIO_Canonicalize_SymlinkChain(t *testing.T) {
	dir := t.TempDir()
	final := filepath.Join(dir, "final.scss")
	if err := os.WriteFile(final, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	link1 := filepath.Join(dir, "chain1")
	link2 := filepath.Join(dir, "chain2.scss")
	if err := os.Symlink(final, link1); err != nil {
		t.Skip("symlinks not supported:", err)
	}
	if err := os.Symlink(link1, link2); err != nil {
		t.Skip("symlinks not supported:", err)
	}

	d := newDefaultIO()
	result, err := d.Canonicalize(link2)
	if err != nil {
		t.Fatal(err)
	}
	// Canonicalize is lexical — it preserves symlink names (matching Dart).
	expected, err := filepath.Abs(link2)
	if err != nil {
		t.Fatal(err)
	}
	if result != expected {
		t.Fatalf("got %q, want %q", result, expected)
	}
}

func TestFileSystemException_Unwrap(t *testing.T) {
	inner := errors.New("inner error")
	fse := &FileSystemException{Message: "test", Path: "/x", Err: inner}
	if !errors.Is(fse, inner) {
		t.Fatal("Unwrap failed")
	}
}

func TestFileSystemException_IsNotExist(t *testing.T) {
	tests := []struct {
		name string
		fse  *FileSystemException
		want bool
	}{
		{"os.ErrNotExist", &FileSystemException{Err: os.ErrNotExist}, true},
		{"nil Err", &FileSystemException{}, false},
		{"random error", &FileSystemException{Err: errors.New("boom")}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fse.IsNotExist(); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileSystemException_Error(t *testing.T) {
	tests := []struct {
		e    *FileSystemException
		want string
	}{
		{&FileSystemException{Message: "err", Path: "/a"}, "err: /a"},
		{&FileSystemException{Message: "err"}, "err"},
	}
	for _, tt := range tests {
		if got := tt.e.Error(); got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
}

func TestFileSystemException_IsPermission(t *testing.T) {
	tests := []struct {
		name string
		fse  *FileSystemException
		want bool
	}{
		{"os.ErrPermission", &FileSystemException{Err: os.ErrPermission}, true},
		{"nil Err", &FileSystemException{}, false},
		{"random error", &FileSystemException{Err: errors.New("boom")}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fse.IsPermission(); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFileSystemException_IsExist(t *testing.T) {
	tests := []struct {
		name string
		fse  *FileSystemException
		want bool
	}{
		{"os.ErrExist", &FileSystemException{Err: os.ErrExist}, true},
		{"nil Err", &FileSystemException{}, false},
		{"random error", &FileSystemException{Err: errors.New("boom")}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fse.IsExist(); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
