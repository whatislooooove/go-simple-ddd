package wiki

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// SectionConfig описывает, как извлечь игры из конкретной секции Wikipedia.
type SectionConfig struct {
	// Label — заголовок, который печатается перед списком игр.
	Label string
	// SectionID — data-mw-section-id основной секции (например "3").
	SectionID string
	// RemoveSubsections — data-mw-section-id подсекций, которые нужно удалить
	// перед парсингом (например "7" — TBA-игры).
	RemoveSubsections []string
	// GameCol — индекс колонки с названием игры (0-based).
	GameCol int
	// PlatformCol — индекс колонки с платформами (0-based).
	PlatformCol int
}

// FetchHTML скачивает страницу и возвращает распарсенный *goquery.Document.
func FetchHTML(url string) (*goquery.Document, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %s", resp.StatusCode)
	}

	html, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(html)))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	return doc, nil
}

// ParseGames извлекает игры из документа по конфигурации секции.
func ParseGames(doc *goquery.Document, cfg SectionConfig) {
	fmt.Println(cfg.Label + "\n")

	mainSection := doc.Find(fmt.Sprintf(`section[data-mw-section-id="%s"]`, cfg.SectionID))

	for _, subID := range cfg.RemoveSubsections {
		mainSection.Find(fmt.Sprintf(`[data-mw-section-id="%s"]`, subID)).Remove()
	}

	// TODO: поправить парсинг игр, которые вышли одного числа - число выхода попадает в строку самой первой игры из нескольких
	mainSection.Find("table.wikitable tbody tr").Each(func(i int, s *goquery.Selection) {
		cols := s.Find("td")
		game := strings.TrimSpace(cols.Eq(cfg.GameCol).Text())
		platforms := strings.TrimSpace(cols.Eq(cfg.PlatformCol).Text())

		if game != "" {
			fmt.Printf("Название: %s\n", game)
			fmt.Printf("Платформы: %s\n", platforms)
			fmt.Println("-----")
		}
	})
}

// FatalError печатает ошибку в stderr и завершает процесс с кодом 1.
func FatalError(err error) {
	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	os.Exit(1)
}
