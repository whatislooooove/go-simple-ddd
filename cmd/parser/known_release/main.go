package main

import (
	"go-ddd-template/cmd/parser/wiki"
)

func main() {
	url := "https://ru.wikipedia.org/wiki/2026_%D0%B3%D0%BE%D0%B4_%D0%B2_%D0%BA%D0%BE%D0%BC%D0%BF%D1%8C%D1%8E%D1%82%D0%B5%D1%80%D0%BD%D1%8B%D1%85_%D0%B8%D0%B3%D1%80%D0%B0%D1%85"

	doc, err := wiki.FetchHTML(url)
	if err != nil {
		wiki.FatalError(err)
	}

	wiki.ParseGames(doc, wiki.KnownRelease)
}
