// internal/feed/bilibili.go
package feed

import (
	"errors"
	"fmt"
	"github.com/glanceapp/glance/internal/parser"
	"log/slog"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

// ParseTimeString 将时间字符串解析为 time.Time
// 支持两种格式： "YYYY-M-D" 和 "M-D"
// 如果是 "M-D" 格式，自动使用当前年份
func ParseTimeString(timeStr string) (time.Time, error) {
	timeStr = strings.TrimSpace(timeStr)

	// 正则表达式匹配完整日期 "YYYY-M-D"
	fullDateRegex := regexp.MustCompile(`^\d{4}-\d{1,2}-\d{1,2}$`)
	// 正则表达式匹配简化日期 "M-D"
	shortDateRegex := regexp.MustCompile(`^\d{1,2}-\d{1,2}$`)

	currentYear := time.Now().Year()

	if fullDateRegex.MatchString(timeStr) {
		// 尝试使用 "2006-1-2" 格式解析
		parsedTime, err := time.Parse("2006-1-2", timeStr)
		if err != nil {
			return time.Time{}, fmt.Errorf("无法解析完整日期格式: %w", err)
		}
		return parsedTime, nil
	} else if shortDateRegex.MatchString(timeStr) {
		// 拼接当前年份，形成 "YYYY-M-D"
		fullDateStr := fmt.Sprintf("%d-%s", currentYear, timeStr)
		parsedTime, err := time.Parse("2006-1-2", fullDateStr)
		if err != nil {
			return time.Time{}, fmt.Errorf("无法解析简化日期格式: %w", err)
		}

		// 如果解析的日期在未来，假设它属于上一年
		if parsedTime.After(time.Now()) {
			fullDateStr = fmt.Sprintf("%d-%s", currentYear-1, timeStr)
			parsedTime, err = time.Parse("2006-1-2", fullDateStr)
			if err != nil {
				return time.Time{}, fmt.Errorf("无法解析简化日期格式为上一年: %w", err)
			}
		}

		return parsedTime, nil
	} else {
		return time.Time{}, errors.New("不支持的日期格式")
	}
}

func FetchBilibiliChannelUploads(channelIds []string, videoUrlTemplate string, includeShorts bool) (Videos, error) {
	var (
		videos     Videos
		fetchError error
	)

	// 定义 XPath 表达式
	cardBoxXPath := "//div[contains(@class, 'content') and contains(@class, 'clearfix')]/div"
	hrefXPath := ".//a[contains(@class, 'cover')]"
	imgXPath := ".//div[contains(@class, 'b-img')]//source[@type='image/webp']"
	titleXPath := ".//a[contains(@class, 'title')]/text()"
	timeXPath := ".//span[contains(@class, 'time')]/text()"
	authorPath := ".//span[@id=\"h-name\"]/text()"

	// 顺序处理每个频道
	for _, channelId := range channelIds {
		channelURL := fmt.Sprintf("https://space.bilibili.com/%s", channelId)
		channelContent := fmt.Sprintf("html/bilibili_%s.html", channelId)

		// 定义调用参数
		options := parser.FetchOptions{
			URL:         channelURL,
			Timeout:     15,
			Show:        true, // 根据需要设置
			Wait:        "load",
			Render:      5000,
			OutputDir:   channelContent,
			UserDataDir: "./user_data",
			CookieFile:  "F:\\glance\\cookie\\bilibili_cookie.json",
			FetcherPath: "F:\\glance\\fetch_web\\fetch_content.exe",
		}

		// 调用 FetchHTML 函数获取 HTML 内容
		htmlContent, err := parser.FetchHTML(options)
		if err != nil {
			slog.Error("Failed to fetch HTML", "channel", channelId, "error", err)
			if fetchError == nil {
				fetchError = err
			}
			continue
		}

		// 解析 HTML 内容
		doc, err := parser.ParseHTMLs(htmlContent)
		if err != nil {
			slog.Error("Failed to parse HTML", "channel", channelId, "error", err)
			if fetchError == nil {
				fetchError = err
			}
			continue
		}

		// 提取作者信息
		author, err := parser.GetText(doc, authorPath)
		if err != nil {
			slog.Error("未能提取 author", "channel", channelId, "error", err)
			author = "None"
		} else {
			author = strings.TrimSpace(author)
		}

		// 查找所有 cardBox 节点
		cardBoxes, err := parser.FindNodes(doc, cardBoxXPath)
		if err != nil {
			slog.Error("Failed to find card boxes", "channel", channelId, "error", err)
			if fetchError == nil {
				fetchError = err
			}
			continue
		}

		var channelVideos Videos

		for _, cardBox := range cardBoxes {
			// 提取 <a class='cover'> 节点并获取 href 属性
			aNode, err := parser.FindNode(cardBox, hrefXPath)
			if err != nil || aNode == nil {
				slog.Error("未找到 a.cover 节点", "channel", channelId, "error", err)
				continue
			}
			href, err := parser.GetAttribute(aNode, "href")
			if err != nil {
				slog.Error("未能提取 href 属性", "channel", channelId, "error", err)
				href = ""
			} else {
				href = strings.Replace(href, "//", "", 1)
			}

			// 提取 <source type='image/webp'> 节点并获取 srcset 属性
			sourceNode, err := parser.FindNode(cardBox, imgXPath)
			if err != nil || sourceNode == nil {
				slog.Error("未找到 source[type='image/webp'] 节点", "channel", channelId, "error", err)
				continue
			}
			imgSrc, err := parser.GetAttribute(sourceNode, "srcset")
			if err != nil {
				slog.Error("未能提取 srcset 属性", "channel", channelId, "error", err)
				imgSrc = ""
			} else {
				imgSrc = "https:" + imgSrc
			}

			// 提取 title
			title, err := parser.GetText(cardBox, titleXPath)
			if err != nil {
				slog.Error("未能提取 title", "channel", channelId, "error", err)
				title = ""
			} else {
				title = strings.TrimSpace(title)
			}

			// 提取 time
			timeStr, err := parser.GetText(cardBox, timeXPath)
			if err != nil {
				slog.Error("未能提取 time", "channel", channelId, "error", err)
				timeStr = ""
			} else {
				timeStr = strings.TrimSpace(timeStr)
			}

			// 解析发布时间
			parsedTime, err := ParseTimeString(timeStr)
			if err != nil {
				slog.Error("无法解析时间字符串", "channel", channelId, "time", timeStr, "error", err)
				parsedTime = time.Time{}
				continue
			}

			// 构造视频 URL
			videoUrl := "https://" + href

			// 构造 Author URL
			authorUrl := fmt.Sprintf("https://space.bilibili.com/%s/video", channelId)

			// 添加视频信息到频道视频列表
			channelVideos = append(channelVideos, Video{
				ThumbnailUrl: imgSrc,
				Title:        title,
				Url:          videoUrl,
				Author:       author,
				AuthorUrl:    authorUrl,
				TimePosted:   parsedTime,
			})
		}

		// 如果没有视频，跳过
		if len(channelVideos) == 0 {
			continue
		}

		// 排序视频，按发布时间降序
		sort.Slice(channelVideos, func(i, j int) bool {
			return channelVideos[i].TimePosted.After(channelVideos[j].TimePosted)
		})

		// 仅保留前 5 个视频
		if len(channelVideos) > 5 {
			channelVideos = channelVideos[:5]
		}

		// 将频道视频添加到总视频列表
		videos = append(videos, channelVideos...)
	}

	if fetchError != nil && len(videos) == 0 {
		return nil, fetchError
	}

	if len(videos) == 0 {
		return nil, ErrNoContent
	}

	return videos, nil
}

// ExtractedData 定义提取的数据结构
type ExtractedData struct {
	Href   string
	ImgSrc string
	Title  string
	Time   string
	Author string
}

// FetchResult 定义抓取结果的结构
type FetchResult struct {
	URL   string
	Data  []ExtractedData
	Error error
}

// extractVideoID 从 href 中提取视频 ID
func extractVideoID(href string) string {
	// 假设 href 格式为 "/video/BV1xxxxxx" 或类似格式，根据实际情况调整
	parts := strings.Split(href, "/")
	if len(parts) > 2 {
		return parts[2]
	}
	return ""
}

// extractAuthorID 从 href 中提取作者 ID
func extractAuthorID(href string) int64 {
	// 假设 href 包含作者 ID，可以根据实际情况提取
	// 这里返回一个固定值作为示例
	return 19956596
}

// addDomain 如果 URL 是相对路径，则添加域名
func addDomain(base, link string) string {
	parsedBase, err := url.Parse(base)
	if err != nil {
		return link
	}
	parsedLink, err := url.Parse(link)
	if err != nil {
		return link
	}
	return parsedBase.ResolveReference(parsedLink).String()
}

// parseBilibiliPubDate 将发布时间字符串转换为 time.Time
func parseBilibiliPubDate(t time.Time) time.Time {
	return t
}
