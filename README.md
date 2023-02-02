# Saggle
Saggle is a toy full text search engine in Golang

You can get wikipedia page titles that contain the word of your query.

Saggle adopts the inverted index to search the word. It will be stored in sqlite3 database at first run.

## Usage
```
$ go run main.go
Enter your query: japan onsen
Wikipedia: Onsen
Wikipedia: Yunokawa Onsen (Hokkaido)
Wikipedia: Shigenobu, Ehime
Wikipedia: Kawauchi, Ehime
Wikipedia: Nakajima, Ehime
Wikipedia: Onsen, Hyōgo
Wikipedia: Arima Onsen
Wikipedia: Unazuki, Toyama
Result: 8 pages found
Enter your query:
```


## Ref.
- https://artem.krylysov.com/blog/2020/07/28/lets-build-a-full-text-search-engine/