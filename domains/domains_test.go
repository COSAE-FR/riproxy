package domains

import (
	"math/rand"
	"testing"
)

func TestSimpleDomain(t *testing.T) {
	tree := New()
	tree.Put("test.exampLe.com")
	if found := tree.Get("test.example.com"); found == false {
		t.Fatalf("Cannot find domain %s", "test.example.com")
	}
	if found := tree.Get("test.example.com."); found == false {
		t.Fatalf("Cannot find domain %s", "test.example.com.")
	}
	if found := tree.Get("sub.test.example.com"); found == true {
		t.Fatalf("Found domain %s", "lol.test.example.com")
	}
	if found := tree.Get("sub.test.example.com."); found == true {
		t.Fatalf("Found domain %s", "lol.test.example.com.")
	}
	if found := tree.Get("example.com"); found == true {
		t.Fatalf("Found domain %s", "example.com")
	}
	if found := tree.Get("example.com."); found == true {
		t.Fatalf("Found domain %s", "example.com.")
	}
}

func TestWildcardDomain(t *testing.T) {
	tree := New()
	tree.Put("*.exampLe.com")
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

func TestIDNASimpleDomain(t *testing.T) {
	tree := NewIDNA()
	tree.Put("test.exampLe.com")
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

func TestIDNAWildcardDomain(t *testing.T) {
	tree := NewIDNA()
	tree.Put("*.exampLe.com")
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

func TestIDNA(t *testing.T) {
	tree := NewIDNA()
	tree.Put("*.éxampLe.com")
	if found := tree.Get("test.éxample.com"); found == false {
		t.Errorf("Cannot find domain %s", "test.éxample.com")
	}
	if found := tree.Get("sub.test.éxample.com"); found == false {
		t.Errorf("Cannot find domain %s", "sub.test.éxample.com")
	}
	if found := tree.Get("éxample.com"); found == true {
		t.Errorf("Found domain %s", "éxample.com")
	}
}

var letters = []rune("aàbcdefghiîjklmnoöpqrstuùvwxyzAÀBCDEFGHIÎJKLMNOÖPQRSTUVWXYZ")
var pathKeys [1000]string // random .paths.of.parts keys
const partsPerKey = 5     // (e.g. .a.b.c)
const bytesPerPart = 15

func randSeq(n int) string {
	b := make([]rune, rand.Intn(n)+2)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func randDomain(partsN, seqN int) string {
	var key string
	for j := 0; j < rand.Intn(partsN)+2; j++ {
		key += "."
		key += randSeq(seqN)
	}
	return key[1:]
}

func init() {
	for i := 0; i < len(pathKeys); i++ {
		pathKeys[i] = randDomain(partsPerKey, bytesPerPart)
	}
}

func BenchmarkNode_PutDomain(b *testing.B) {
	for i := 0; i < b.N; i++ {
		d := New()
		for _, s := range pathKeys {
			d.Put(s)
		}
	}
}

func BenchmarkIDNANode_PutDomain(b *testing.B) {
	for i := 0; i < b.N; i++ {
		d := NewIDNA()
		for _, s := range pathKeys {
			d.Put(s)
		}
	}
}

func BenchmarkNewFromList(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewFromList(pathKeys[:])
	}
}

func BenchmarkNewIDNAFromList(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewIDNAFromList(pathKeys[:])
	}
}

func BenchmarkNode_Get(b *testing.B) {
	tree := NewFromList(pathKeys[:])
	needle := pathKeys[rand.Intn(len(pathKeys))]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tree.Get(needle)
	}
}

func BenchmarkIDNANode_Get(b *testing.B) {
	tree := NewIDNAFromList(pathKeys[:])
	needle := pathKeys[rand.Intn(len(pathKeys))]
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tree.Get(needle)
	}
}

func BenchmarkNode_NoGet(b *testing.B) {
	tree := NewFromList(pathKeys[:])
	needle := randDomain(partsPerKey, bytesPerPart)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tree.Get(needle)
	}
}

func BenchmarkIDNANode_NoGet(b *testing.B) {
	tree := NewIDNAFromList(pathKeys[:])
	needle := randDomain(partsPerKey, bytesPerPart)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = tree.Get(needle)
	}
}
