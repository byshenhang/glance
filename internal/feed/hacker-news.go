package feed

import (
	"fmt"
	"github.com/glanceapp/glance/internal/tool"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type hackerNewsPostResponseJson struct {
	Id           int    `json:"id"`
	Score        int    `json:"score"`
	Title        string `json:"title"`
	TargetUrl    string `json:"url,omitempty"`
	CommentCount int    `json:"descendants"`
	TimePosted   int64  `json:"time"`
}

func getHackerNewsPostIds(sort string) ([]int, error) {
	request, _ := http.NewRequest("GET", fmt.Sprintf("https://hacker-news.firebaseio.com/v0/%sstories.json", sort), nil)
	response, err := decodeJsonFromRequest[[]int](defaultClient, request)

	if err != nil {
		return nil, fmt.Errorf("%w: could not fetch list of post IDs", ErrNoContent)
	}

	return response, nil
}

// translationResult holds the result of a translation task
type translationResult struct {
	Index           int
	OriginalTitle   string
	TranslatedTitle string
	Err             error
}

// translateTitlesConcurrently 并发翻译多个标题
func translateTitlesConcurrently(titles []string) []string {
	var wg sync.WaitGroup
	resultsChan := make(chan translationResult, len(titles))
	concurrencyLimit := 2 // 最大并发数，根据实际情况调整

	sem := make(chan struct{}, concurrencyLimit)

	for i, title := range titles {
		wg.Add(1)
		go func(index int, text string) {
			defer wg.Done()
			sem <- struct{}{}        // 获取一个信号量
			defer func() { <-sem }() // 释放信号量

			translated, err := tool.PromptTranslate(text)
			if err != nil {
				slog.Error("Failed to tool title", "error", err, "title", text)
				translated = text // 翻译失败，使用原始标题
			}

			resultsChan <- translationResult{
				Index:           index,
				OriginalTitle:   text,
				TranslatedTitle: translated,
				Err:             err,
			}
		}(i, title)
	}

	// 等待所有翻译完成
	wg.Wait()
	close(resultsChan)

	// 收集结果
	translatedTitles := make([]string, len(titles))
	for res := range resultsChan {
		translatedTitles[res.Index] = res.TranslatedTitle
	}

	return translatedTitles
}

func getHackerNewsPostsFromIds(postIds []int, commentsUrlTemplate string) (ForumPosts, error) {
	requests := make([]*http.Request, len(postIds))

	for i, id := range postIds {
		request, _ := http.NewRequest("GET", fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", id), nil)
		requests[i] = request
	}

	task := decodeJsonFromRequestTask[hackerNewsPostResponseJson](defaultClient)
	job := newJob(task, requests).withWorkers(30)
	results, errs, err := workerPoolDo(job)

	if err != nil {
		return nil, err
	}

	// 收集所有需要翻译的标题
	titles := make([]string, 0, len(postIds))
	indexMap := make([]int, 0, len(postIds)) // 保存每个标题对应的原始索引

	for i, res := range results {
		if errs[i] != nil {
			slog.Error("Failed to fetch or parse hacker news post", "error", errs[i], "url", requests[i].URL)
			continue
		}
		titles = append(titles, res.Title)
		indexMap = append(indexMap, i)
	}

	// 并发翻译标题
	translatedTitles := translateTitlesConcurrently(titles)

	// 构建 ForumPosts
	posts := make(ForumPosts, 0, len(postIds))

	for i, translatedTitle := range translatedTitles {
		originalIndex := indexMap[i]
		res := results[originalIndex]

		var commentsUrl string
		if commentsUrlTemplate == "" {
			commentsUrl = "https://news.ycombinator.com/item?id=" + strconv.Itoa(res.Id)
		} else {
			commentsUrl = strings.ReplaceAll(commentsUrlTemplate, "{POST-ID}", strconv.Itoa(res.Id))
		}

		posts = append(posts, ForumPost{
			Title:           translatedTitle,
			DiscussionUrl:   commentsUrl,
			TargetUrl:       res.TargetUrl,
			TargetUrlDomain: extractDomainFromUrl(res.TargetUrl),
			CommentCount:    res.CommentCount,
			Score:           res.Score,
			TimePosted:      time.Unix(res.TimePosted, 0),
		})
	}

	if len(posts) == 0 {
		return nil, ErrNoContent
	}

	if len(posts) != len(postIds) {
		return posts, fmt.Errorf("%w could not fetch some hacker news posts", ErrPartialContent)
	}

	return posts, nil
}

func FetchHackerNewsPosts(sort string, limit int, commentsUrlTemplate string) (ForumPosts, error) {
	postIds, err := getHackerNewsPostIds(sort)

	if err != nil {
		return nil, err
	}

	if len(postIds) > limit {
		postIds = postIds[:limit]
	}

	return getHackerNewsPostsFromIds(postIds, commentsUrlTemplate)
}
