package domains

import "testing"

func TestSimpleDomain(t *testing.T) {
	tree := New()
	tree.Put("test.example.com")
	if found := tree.Get("test.example.com"); found == false {
		t.Fatalf("Cannot find domain %s", "test.example.com")
	}
	if found := tree.Get("lol.test.example.com"); found == true {
		t.Fatalf("Found domain %s", "lol.test.example.com")
	}
	if found := tree.Get("example.com"); found == true {
		t.Fatalf("Found domain %s", "example.com")
	}
}

func TestWildcardDomain(t *testing.T) {
	tree := New()
	tree.Put("*.example.com")
	if found := tree.Get("test.example.com"); found == false {
		t.Fatalf("Cannot find domain %s", "test.example.com")
	}
	if found := tree.Get("sub.test.example.com"); found == false {
		t.Fatalf("Cannot find domain %s", "sub.test.example.com")
	}
	if found := tree.Get("example.com"); found == true {
		t.Fatalf("Found domain %s", "example.com")
	}
}
