package sassurl

import (
	"encoding/json"
	"os"
	"testing"
)

// These vectors are generated from Dart 3.13.5 (see fixtures/meta.json) and
// cover Uri.parse/resolve, package:path toUri/fromUri, and Uri.dataFromString.
// They are the conformance gate for the Dart-compatible URL implementation.

type dartURICase struct {
	Input           string  `json:"input"`
	OK              bool    `json:"ok"`
	Error           string  `json:"error"`
	Scheme          string  `json:"scheme"`
	UserInfo        string  `json:"userInfo"`
	Host            string  `json:"host"`
	HasAuthority    bool    `json:"hasAuthority"`
	Port            *int    `json:"port"`
	Path            string  `json:"path"`
	HasAbsolutePath bool    `json:"hasAbsolutePath"`
	Query           *string `json:"query"`
	Fragment        *string `json:"fragment"`
	IsAbsolute      bool    `json:"isAbsolute"`
	ToString        string  `json:"toString"`
}

type dartResolveCase struct {
	Base string `json:"base"`
	Ref  string `json:"ref"`
	dartURICase
}

type dartPathCase struct {
	Windows       bool    `json:"windows"`
	Path          string  `json:"path"`
	ToURI         string  `json:"toUri"`
	ToURIError    *string `json:"toUriError"`
	FromURI       string  `json:"fromUri"`
	FromURIError  *string `json:"fromUriError"`
	FileURI       string  `json:"fileUri"`
	FileURIError  *string `json:"fileUriError"`
	FilePath      string  `json:"filePath"`
	FilePathError *string `json:"filePathError"`
}

type dartDataCase struct {
	Contents string `json:"contents"`
	Data     string `json:"data"`
}

func loadFixture(t *testing.T, name string, dst any) {
	t.Helper()
	b, err := os.ReadFile("fixtures/" + name + ".json")
	if err != nil {
		t.Fatalf("read fixtures/%s.json: %v", name, err)
	}
	if err := json.Unmarshal(b, dst); err != nil {
		t.Fatalf("parse fixtures/%s.json: %v", name, err)
	}
}

func checkDartURI(t *testing.T, label string, u *URL, c *dartURICase) {
	t.Helper()
	if u.Scheme != c.Scheme {
		t.Fatalf("%s: scheme = %q, want %q", label, u.Scheme, c.Scheme)
	}
	if u.UserInfo != c.UserInfo {
		t.Fatalf("%s: userInfo = %q, want %q", label, u.UserInfo, c.UserInfo)
	}
	if u.Host != c.Host {
		t.Fatalf("%s: host = %q, want %q", label, u.Host, c.Host)
	}
	if u.HasAuthority != c.HasAuthority {
		t.Fatalf("%s: hasAuthority = %v, want %v", label, u.HasAuthority, c.HasAuthority)
	}
	if c.Port == nil {
		if u.HasPort {
			t.Fatalf("%s: hasPort = true, want false", label)
		}
	} else if !u.HasPort || u.Port != *c.Port {
		t.Fatalf("%s: port = %d/%v, want %d", label, u.Port, u.HasPort, *c.Port)
	}
	if u.Path != c.Path {
		t.Fatalf("%s: path = %q, want %q", label, u.Path, c.Path)
	}
	if u.HasAbsolutePath() != c.HasAbsolutePath {
		t.Fatalf("%s: hasAbsolutePath = %v, want %v", label, u.HasAbsolutePath(), c.HasAbsolutePath)
	}
	if c.Query == nil {
		if u.HasQuery {
			t.Fatalf("%s: hasQuery = true, want false", label)
		}
	} else if !u.HasQuery || u.Query != *c.Query {
		t.Fatalf("%s: query = %q, want %q", label, u.Query, *c.Query)
	}
	if c.Fragment == nil {
		if u.HasFragment {
			t.Fatalf("%s: hasFragment = true, want false", label)
		}
	} else if !u.HasFragment || u.Fragment != *c.Fragment {
		t.Fatalf("%s: fragment = %q, want %q", label, u.Fragment, *c.Fragment)
	}
	if u.IsAbs() != c.IsAbsolute {
		t.Fatalf("%s: isAbsolute = %v, want %v", label, u.IsAbs(), c.IsAbsolute)
	}
	if u.String() != c.ToString {
		t.Fatalf("%s: toString = %q, want %q", label, u.String(), c.ToString)
	}
}

func TestDartURIParseFixtures(t *testing.T) {
	var cases []dartURICase
	loadFixture(t, "parse", &cases)
	for i := range cases {
		c := &cases[i]
		u, err := Parse(c.Input)
		if !c.OK {
			if err == nil {
				t.Fatalf("parse %d %q: got %q, want error %q", i, c.Input, u.String(), c.Error)
			}
			if err.Error() != c.Error {
				t.Fatalf("parse %d %q: error = %q, want %q", i, c.Input, err.Error(), c.Error)
			}
			continue
		}
		if err != nil {
			t.Fatalf("parse %d %q: %v", i, c.Input, err)
		}
		checkDartURI(t, "parse "+c.Input, u, c)
	}
}

func TestDartURIResolveFixtures(t *testing.T) {
	var cases []dartResolveCase
	loadFixture(t, "resolve", &cases)
	for i := range cases {
		c := &cases[i]
		base, err := Parse(c.Base)
		if err != nil {
			t.Fatalf("resolve %d: base %q: %v", i, c.Base, err)
		}
		ref, err := Parse(c.Ref)
		if err != nil {
			t.Fatalf("resolve %d: ref %q: %v", i, c.Ref, err)
		}
		checkDartURI(t, "resolve "+c.Base+" + "+c.Ref, base.Resolve(ref), &c.dartURICase)
	}
}

func TestDartURIPathFixtures(t *testing.T) {
	var cases []dartPathCase
	loadFixture(t, "path", &cases)
	for i := range cases {
		c := &cases[i]

		toURI, toURIErr := pathToURIWithOS(c.Path, c.Windows)
		if c.ToURIError != nil {
			if toURIErr == nil {
				t.Fatalf("path %d %q: toUri = %q, want error", i, c.Path, toURI.String())
			}
		} else {
			if toURIErr != nil {
				t.Fatalf("path %d %q: toUri error: %v", i, c.Path, toURIErr)
			}
			if toURI.String() != c.ToURI {
				t.Fatalf("path %d %q: toUri = %q, want %q", i, c.Path, toURI.String(), c.ToURI)
			}
			fromURI, ok := uriToPath(toURI.Path, hostPtr(toURI), c.Windows)
			if c.FromURIError != nil {
				if ok {
					t.Fatalf("path %d %q: fromUri = %q, want error", i, c.Path, fromURI)
				}
			} else if !ok || fromURI != c.FromURI {
				t.Fatalf("path %d %q: fromUri = %q/%v, want %q", i, c.Path, fromURI, ok, c.FromURI)
			}
		}

		fileURI, fileURIErr := uriFile(preparePath(c.Path, c.Windows), c.Windows)
		if c.FileURIError != nil {
			if fileURIErr == nil {
				t.Fatalf("path %d %q: fileUri = %q, want error", i, c.Path, fileURI.String())
			}
		} else {
			if fileURIErr != nil {
				t.Fatalf("path %d %q: fileUri error: %v", i, c.Path, fileURIErr)
			}
			if fileURI.String() != c.FileURI {
				t.Fatalf("path %d %q: fileUri = %q, want %q", i, c.Path, fileURI.String(), c.FileURI)
			}
			filePath, ok := uriToPath(fileURI.Path, hostPtr(fileURI), c.Windows)
			if c.FilePathError != nil {
				if ok {
					t.Fatalf("path %d %q: filePath = %q, want error", i, c.Path, filePath)
				}
			} else if !ok || filePath != c.FilePath {
				t.Fatalf("path %d %q: filePath = %q/%v, want %q", i, c.Path, filePath, ok, c.FilePath)
			}
		}
	}
}

func TestDartURIDataFixtures(t *testing.T) {
	var cases []dartDataCase
	loadFixture(t, "data", &cases)
	for i := range cases {
		c := &cases[i]
		if got := DataURLFromString(c.Contents); got != c.Data {
			t.Fatalf("data %d: got %q, want %q", i, got, c.Data)
		}
	}
}
