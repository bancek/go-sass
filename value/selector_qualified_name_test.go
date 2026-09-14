package value

import "testing"

func TestNewQualifiedName(t *testing.T) {
	q := NewQualifiedName("div")
	if q.Name != "div" {
		t.Errorf("Name = %q, want div", q.Name)
	}
	if q.Namespace != nil {
		t.Error("Namespace should be nil")
	}
}

func TestNewQualifiedNameWithNamespace(t *testing.T) {
	ns := "svg"
	q := NewQualifiedNameWithNamespace("circle", &ns)
	if q.Name != "circle" {
		t.Errorf("Name = %q, want circle", q.Name)
	}
	if q.Namespace == nil || *q.Namespace != "svg" {
		t.Error("Namespace should be 'svg'")
	}
}

func TestQualifiedNameWithEmptyNamespace(t *testing.T) {
	ns := ""
	q := NewQualifiedNameWithNamespace("test", &ns)
	if q.Namespace == nil {
		t.Error("Namespace should be non-nil")
	}
	if *q.Namespace != "" {
		t.Errorf("Namespace = %q, want empty string", *q.Namespace)
	}
}

func TestQualifiedNameEqual(t *testing.T) {
	q1 := NewQualifiedName("div")
	q2 := NewQualifiedName("div")
	if !q1.Equal(q2) {
		t.Error("Same name, both nil namespace: should be equal")
	}

	q3 := NewQualifiedName("span")
	if q1.Equal(q3) {
		t.Error("Different names: should not be equal")
	}

	ns := "svg"
	q4 := NewQualifiedNameWithNamespace("div", &ns)
	if q1.Equal(q4) {
		t.Error("One nil namespace, one non-nil: should not be equal")
	}

	ns2 := "svg"
	q5 := NewQualifiedNameWithNamespace("div", &ns2)
	if !q4.Equal(q5) {
		t.Error("Same name, same namespace: should be equal")
	}

	ns3 := "html"
	q6 := NewQualifiedNameWithNamespace("div", &ns3)
	if q4.Equal(q6) {
		t.Error("Same name, different namespace: should not be equal")
	}
}

func TestQualifiedNameEqualBothEmptyNamespace(t *testing.T) {
	empty1 := ""
	empty2 := ""
	q1 := NewQualifiedNameWithNamespace("div", &empty1)
	q2 := NewQualifiedNameWithNamespace("div", &empty2)
	if !q1.Equal(q2) {
		t.Error("Same name, both empty namespace: should be equal")
	}
}

func TestQualifiedNameEqualDifferentNamesSameEmptyNS(t *testing.T) {
	empty := ""
	q1 := NewQualifiedNameWithNamespace("div", &empty)
	q2 := NewQualifiedNameWithNamespace("span", &empty)
	if q1.Equal(q2) {
		t.Error("Different names, same empty namespace: should not be equal")
	}
}

func TestQualifiedNameString(t *testing.T) {
	q := NewQualifiedName("div")
	if s := q.String(); s != "div" {
		t.Errorf("String() = %q, want div", s)
	}

	ns := "svg"
	q2 := NewQualifiedNameWithNamespace("circle", &ns)
	if s := q2.String(); s != "svg|circle" {
		t.Errorf("String() = %q, want svg|circle", s)
	}

	ns3 := "*"
	q3 := NewQualifiedNameWithNamespace("div", &ns3)
	if s := q3.String(); s != "*|div" {
		t.Errorf("String() = %q, want *|div", s)
	}

	ns4 := ""
	q4 := NewQualifiedNameWithNamespace("div", &ns4)
	if s := q4.String(); s != "|div" {
		t.Errorf("String() = %q, want |div", s)
	}
}

func TestQualifiedNameHashCode(t *testing.T) {
	q1 := NewQualifiedName("div")
	q2 := NewQualifiedName("div")
	if q1.HashCode() != q2.HashCode() {
		t.Error("Same name: hash codes should be equal")
	}

	q3 := NewQualifiedName("span")
	if q1.HashCode() == q3.HashCode() {
		t.Error("Different names: hash codes should (likely) differ")
	}

	ns := "svg"
	q4 := NewQualifiedNameWithNamespace("div", &ns)
	q5 := NewQualifiedNameWithNamespace("div", &ns)
	if q4.HashCode() != q5.HashCode() {
		t.Error("Same name + namespace: hash codes should be equal")
	}
}
