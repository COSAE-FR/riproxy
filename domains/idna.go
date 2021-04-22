package domains

import (
	"golang.org/x/net/idna"
	"strings"
	"unicode/utf8"
)

const lower = 'a' - 'A'

func isASCIIandLower(s string) (string, bool) {
	isASCII, hasUpper := true, false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= utf8.RuneSelf {
			isASCII = false
			break
		}
		hasUpper = hasUpper || ('A' <= c && c <= 'Z')
	}

	if isASCII { // optimize for ASCII-only strings.
		if !hasUpper {
			return s, true
		}
		var b strings.Builder
		b.Grow(len(s))
		for i := 0; i < len(s); i++ {
			c := s[i]
			if 'A' <= c && c <= 'Z' {
				c += lower
			}
			b.WriteByte(c)
		}
		return b.String(), true
	}
	return s, false
}

func idnaFormatter(key string) string {
	key, ascii := isASCIIandLower(key)
	if !ascii { // Small optimization, IDN domains are uncommon for now
		var err error
		var idn string
		// Remove leading wildcard for IDN normalization
		if len(key) > 1 && key[0] == '*' && key[1] == '.' {
			idn, err = idna.Lookup.ToASCII(key[2:])
			if err != nil {
				idn = "*." + idn
			}
		} else {
			idn, err = idna.Lookup.ToASCII(key)
		}
		if err != nil {
			return strings.ToLower(idn)
		}
	}
	return key
}

func NewIDNA() DomainTree {
	return &node{
		formatter: idnaFormatter,
	}
}

func NewIDNAFromList(list []string) DomainTree {
	if list != nil && len(list) > 0 {
		node := NewIDNA()
		for _, domain := range list {
			node.Put(domain)
		}
		return node
	}
	return nil
}
