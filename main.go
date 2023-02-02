package main

import (
	"bufio"
	"database/sql"
	"encoding/xml"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"log"
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

func search_contain(docs []document, term string) []document {
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
	"is": {}, "it": {}, "for": {}, "an": {}, "as": {},
	"at": {}, "by": {}, "from": {}, "he": {}, "on": {},
	"or": {}, "this": {}, "was": {}, "were": {},
	"with": {}, "are": {}, "but": {}, "not": {},
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

func (idx index) search(text string) []int {
	var r []int
	for _, token := range analyze(text) {
		if ids, ok := idx[token]; ok {
			if r == nil {
				r = ids
			} else {
				r = intersection(r, ids)
			}
		} else {
			// U \intersection φ = φ
			return nil
		}
	}
	return r
}

// a and b is expected to be sorted.
func intersection(a []int, b []int) []int {
	maxLen := len(a)
	if len(b) > maxLen {
		maxLen = len(b)
	}
	r := make([]int, 0, maxLen)
	var i, j int
	for i < len(a) && j < len(b) {
		if a[i] < b[j] {
			i++
		} else if a[i] > b[j] {
			j++
		} else {
			r = append(r, a[i])
			i++
			j++
		}
	}
	return r
}

func main() {
	docs, err := loadDocuments("enwiki-latest-abstract1.xml")
	if err != nil {
		fmt.Println(err)
	}
	// if db doesnot exists, create index
	idx := make(index)
	if _, err := os.Stat("./index.db"); os.IsNotExist(err) {
		idx.add(docs)
		create_index_db(idx)
	}
	for {
		fmt.Print("Enter your query: ")
		var s string
		r := bufio.NewReader(os.Stdin)
		s, _ = r.ReadString('\n')
		count := 0
		for _, id := range search_from_db(s) {
			fmt.Println(docs[id].Title)
			count += 1

		}
		fmt.Printf("Result: %v pages found\n", count)
	}
}
func search_from_db(query string) []int {

	db, err := sql.Open("sqlite3", "./index.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	var r []int
	for _, token := range analyze(query) {
		rows, err := db.Query("select doc_id from index_table where token = ?", token)
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()
		var ids []int
		for rows.Next() {
			var doc_id int
			err = rows.Scan(&doc_id)
			if err != nil {
				log.Fatal(err)
			}
			ids = append(ids, doc_id)
		}
		err = rows.Err()
		if err != nil {
			log.Fatal(err)
		}
		if r == nil {
			r = ids
		} else {
			r = intersection(r, ids)
		}
	}
	return r
}

func create_index_db(idx index) {
	os.Remove("./index.db")
	db, err := sql.Open("sqlite3", "./index.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	sqlStmt := `
	create table index_table (token text not null, doc_id integer);
	delete from index_table;
	`
	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	tx, err := db.Begin()
	if err != nil {
		log.Fatal(err)
	}
	stmt, err := tx.Prepare("insert into index_table(token, doc_id) values(?, ?)")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	for token, doc_ids := range idx {
		for _, doc_id := range doc_ids {
			_, err = stmt.Exec(token, doc_id)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	err = tx.Commit()
	if err != nil {
		log.Fatal(err)
	}

	rows, err := db.Query("select token, doc_id from index_table")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var token string
		var doc_id int
		err = rows.Scan(&token, &doc_id)
		if err != nil {
			log.Fatal(err)
		}
	}
	err = rows.Err()
	if err != nil {
		log.Fatal(err)
	}

	stmt, err = db.Prepare("select doc_id from index_table where token=?")
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()
	var doc_id []int
	err = stmt.QueryRow("cat").Scan(&doc_id)
	if err != nil {
		log.Fatal(err)
	}
}
