package similarity

import (
	"hash/fnv"
	"sort"
	"strings"
	"unicode"
)

const (
	kgram = 6
	window = 4
	minPrints = 8
)

var keep = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`if else for while do switch case default break continue return int long short double float char bool void
		unsigned signed const auto struct class true false new delete sizeof using namespace std cout cin endl string vector getline
		main include iostream cmath sqrt pow abs max min swap sort size push_back length`) {
		keep[w] = true
	}
}

func Tokenize(code string) []string {
	var out []string
	r := []rune(code)
	for i := 0; i < len(r); {
		c := r[i]
		switch {
		case unicode.IsSpace(c):
			i++
		case c == '#':
			for i < len(r) && r[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(r) && r[i+1] == '/':
			for i < len(r) && r[i] != '\n' {
				i++
			}
		case c == '/' && i+1 < len(r) && r[i+1] == '*':
			i += 2
			for i+1 < len(r) && !(r[i] == '*' && r[i+1] == '/') {
				i++
			}
			i += 2
		case c == '"' || c == '\'':
			quote := c
			i++
			for i < len(r) && r[i] != quote && r[i] != '\n' {
				if r[i] == '\\' {
					i++
				}
				i++
			}
			i++
			if quote == '"' {
				out = append(out, "S")
			} else {
				out = append(out, "C")
			}
		case unicode.IsDigit(c):
			for i < len(r) && (unicode.IsDigit(r[i]) || unicode.IsLetter(r[i]) || r[i] == '.') {
				i++
			}
			out = append(out, "N")
		case unicode.IsLetter(c) || c == '_':
			j := i
			for j < len(r) && (unicode.IsLetter(r[j]) || unicode.IsDigit(r[j]) || r[j] == '_') {
				j++
			}
			word := string(r[i:j])
			if keep[word] {
				out = append(out, word)
			} else {
				out = append(out, "V")
			}
			i = j
		default:
			if i+1 < len(r) && strings.Contains("<< >> == != <= >= && || ++ -- += -= *= /= %= ::", string(r[i:i+2])) {
				out = append(out, string(r[i:i+2]))
				i += 2
			} else {
				out = append(out, string(c))
				i++
			}
		}
	}
	return out
}

func Fingerprints(tokens []string) map[uint64]bool {
	prints := map[uint64]bool{}
	if len(tokens) < kgram {
		return prints
	}
	hashes := make([]uint64, len(tokens)-kgram+1)
	for i := range hashes {
		h := fnv.New64a()
		h.Write([]byte(strings.Join(tokens[i:i+kgram], " ")))
		hashes[i] = h.Sum64()
	}
	if len(hashes) <= window {
		for _, h := range hashes {
			prints[h] = true
		}
		return prints
	}
	for i := 0; i+window <= len(hashes); i++ {
		best := hashes[i]
		for _, h := range hashes[i+1 : i+window] {
			if h < best {
				best = h
			}
		}
		prints[best] = true
	}
	return prints
}

type Pair struct {
	A string `json:"a"`
	B string `json:"b"`
	Score float64 `json:"score"`
}

func Compare(docs map[string]string, minScore float64) []Pair {
	prints := map[string]map[uint64]bool{}
	common := map[uint64]int{}
	for id, code := range docs {
		p := Fingerprints(Tokenize(code))
		prints[id] = p
		for h := range p {
			common[h]++
		}
	}
	if len(docs) >= 4 {
		for id, p := range prints {
			kept := map[uint64]bool{}
			for h := range p {
				if common[h]*2 <= len(docs) {
					kept[h] = true
				}
			}
			prints[id] = kept
		}
	}
	ids := make([]string, 0, len(docs))
	for id, p := range prints {
		if len(p) >= minPrints {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	var out []Pair
	for i, a := range ids {
		for _, b := range ids[i+1:] {
			shared := 0
			for h := range prints[a] {
				if prints[b][h] {
					shared++
				}
			}
			union := len(prints[a]) + len(prints[b]) - shared
			if score := float64(shared) / float64(union); score >= minScore {
				out = append(out, Pair{A: a, B: b, Score: score})
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	return out
}
