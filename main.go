package main

import (
	"encoding/xml"
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"

	snowballeng "github.com/kljensen/snowball/english"
)

type document struct {
	Title string `xml:"title"`
	URL   string `xml:"url"`
	Text  string `xml:"abstract"`
	ID    int
}

func loadDocuments(path string) ([]document, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := xml.NewDecoder(f)
	dump := struct {
		Documents []document `xml:"doc"`
	}{}
	if err := dec.Decode(&dump); err != nil {
		return nil, err
	}
	docs := dump.Documents
	for i := range docs {
		docs[i].ID = i
	}
	return docs, nil
}

func search(docs []document, term string) []document {
	var r []document
	for _, doc := range docs {
		if strings.Contains(doc.Text, term) {
			r = append(r, doc)
		}
	}
	return r
}
func search_re(docs []document, term string) []document {
	re := regexp.MustCompile(`(?i)\b` + term + `\b`)
	var r []document
	for _, doc := range docs {
		if re.MatchString(doc.Text) {
			r = append(r, doc)
		}
	}
	return r
}

func tokenize(text string) []string {
	return strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r)
	})
}

func lowercaseFilter(tokens []string) []string {
	r := make([]string, len(tokens))
	for i, token := range tokens {
		r[i] = strings.ToLower(token)
	}
	return r
}

var stopwords = map[string]struct{}{
	"a": {}, "and": {}, "be": {}, "have": {}, "i": {},
	"in": {}, "of": {}, "that": {}, "the": {}, "to": {},
}

func stopwordFilter(tokens []string) []string {
	r := make([]string, 0, len(tokens))
	for _, token := range tokens {
		if _, ok := stopwords[token]; !ok {
			r = append(r, token)
		}
	}
	return r
}

func stemmerFilter(tokens []string) []string {
	r := make([]string, len(tokens))
	for i, token := range tokens {
		r[i] = snowballeng.Stem(token, false)
	}
	return r
}

func analyze(text string) []string {
	tokens := tokenize(text)
	tokens = lowercaseFilter(tokens)
	tokens = stopwordFilter(tokens)
	tokens = stemmerFilter(tokens)
	return tokens
}

type index map[string][]int

func (idx index) add(docs []document) {
	for _, doc := range docs {
		for _, token := range analyze(doc.Text) {
			ids := idx[token]
			if ids != nil && ids[len(ids)-1] == doc.ID {
				continue
			}
			idx[token] = append(ids, doc.ID)
		}
	}
}

func main() {
	//docs, err := loadDocuments("enwiki-latest-abstract1.xml")
	//if err != nil {
	//	fmt.Println(err)
	//}
	//r := search_re(docs, "cat")
	//for _, doc := range r {
	//	fmt.Println(doc.Title)
	//}
	text := "A donut on a glass plate. Only the donuts."
	tokens := tokenize(text)
	fmt.Println(tokens)
	lower_tokens := lowercaseFilter(tokens)
	fmt.Println(lower_tokens)
	stop_word_removed_tokens := stopwordFilter(lower_tokens)
	fmt.Println(stop_word_removed_tokens)
	stem_tokens := stemmerFilter(stop_word_removed_tokens)
	fmt.Println(stem_tokens)
	analyzed_tokens := analyze(text)
	fmt.Printf("%v\n", analyzed_tokens)
	idx := make(index)
	idx.add([]document{{ID: 1, Text: "A donut on a glass plate. Only the donuts."}})
	idx.add([]document{{ID: 2, Text: "donut is a donut"}})
	fmt.Println(idx)
}
