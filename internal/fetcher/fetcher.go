package fetcher

import (
	"context"
	"log"
	"strings"
	"sync"
	"telebot/internal/model"
	"telebot/source"
	"time"

	"github.com/tomakado/containers/set"
)

type ArticleStorage interface {
	Store(ctx context.Context, article model.Article) error
}

type SourceProvider interface {
	Sources(ctx context.Context) ([]model.Source, error)
}

type Source interface {
	ID() int64
	Name() string
	Fetch(ctx context.Context) ([]model.Item, error)
}

func New(
	articleStorage ArticleStorage,
	sourceProvider SourceProvider,
	fetchInterval time.Duration,
	filterKeywords []string,
) *Fetcher {
	return &Fetcher{
		arcticles:      articleStorage,
		sources:        sourceProvider,
		fetchInterval:  fetchInterval,
		filterKeywords: filterKeywords,
	}
}

type Fetcher struct {
	arcticles ArticleStorage
	sources   SourceProvider

	fetchInterval  time.Duration
	filterKeywords []string
}

func (f *Fetcher) Fetch(ctx context.Context) error {
	sources, err := f.sources.Sources(ctx)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup

	for _, src := range sources {
		wg.Add(1)

		rssSource := source.NewRSSSourceFromModel(src)

		go func(source Source) {
			defer wg.Done()

			items, err := source.Fetch(ctx)
			if err != nil {
				log.Printf("[ERROR] Fetching items from source %s: %v", source.Name(), err)
			}

			if err := f.processItems(ctx, source, items); err != nil {
				log.Printf("[ERROR] Processing items from source %s: %v", source.Name(), err)
			}
		}(rssSource)
	}

	wg.Wait()

	return nil
}

func (f *Fetcher) processItems(ctx context.Context, source Source, items []model.Item) error {
	for _, item := range items {
		item.Date = item.Date.UTC()

		// если item есть в ключах или загловке
		// пропускаем статью
		if f.itemShouldBeSkipped(item) {
			continue
		}

		if err := f.arcticles.Store(ctx, model.Article{
			SourceID:    source.ID(),
			Title:       item.Title,
			Link:        item.Link,
			Summary:     item.Summary,
			PublishedAt: item.Date,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (f *Fetcher) itemShouldBeSkipped(item model.Item) bool {
	// создадим хеш-сет
	categoriesSet := set.New(item.Categories...)

	for _, keyword := range f.filterKeywords {
		titleContainsKeyword := strings.Contains(strings.ToLower(item.Title), keyword)
		if categoriesSet.Contains(keyword) || titleContainsKeyword {
			return true
		}
	}
	return false
}
