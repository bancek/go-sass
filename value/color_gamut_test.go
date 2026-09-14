package value

import "testing"

func TestGamutMapMethodFromName(t *testing.T) {
	m, err := GamutMapMethodFromName("clip")
	if err != nil {
		t.Fatal(err)
	}
	if m != GamutMapClip {
		t.Error("should be clip")
	}
	m2, err := GamutMapMethodFromName("local-minde")
	if err != nil {
		t.Fatal(err)
	}
	if m2 != GamutMapLocalMinde {
		t.Error("should be local-minde")
	}
	_, err = GamutMapMethodFromName("bogus")
	if err == nil {
		t.Error("bogus should error")
	}
}

func TestGamutMapMethodString(t *testing.T) {
	if GamutMapClip.String() != "clip" {
		t.Error()
	}
	if GamutMapLocalMinde.String() != "local-minde" {
		t.Error()
	}
}
