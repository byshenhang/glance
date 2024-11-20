package main

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"

	"parsehtml/parser" // 根据您的模块名调整导入路径
)

// ExtractedData 定义提取的数据结构
type ExtractedData struct {
	Href   string
	ImgSrc string
	Title  string
	Time   string
}

// FetchResult 定义抓取结果的结构
type FetchResult struct {
	URL   string
	Data  []ExtractedData
	Error error
}

func main() {
	// 定义多个 URL
	urls := []string{
		"https://space.bilibili.com/19956596",
	}

	// 定义 XPath 表达式，用于查找所有目标 <div> 标签的节点
	cardBoxXPath := "//div[contains(@class, 'content') and contains(@class, 'clearfix')]/div"

	// 在每个 cardBox 中定义具体的字段 XPath（仅选择节点）
	hrefXPath := ".//a[contains(@class, 'cover')]"
	imgXPath := ".//div[contains(@class, 'b-img')]//source[@type='image/webp']"
	titleXPath := ".//a[contains(@class, 'title')]/text()"
	timeXPath := ".//span[contains(@class, 'time')]/text()"

	// 使用 WaitGroup 进行并发处理
	var wg sync.WaitGroup
	resultsChan := make(chan FetchResult, len(urls))

	for _, baseURL := range urls {
		wg.Add(1)
		go func(baseURL string) {
			defer wg.Done()

			// 定义调用参数
			options := parser.FetchOptions{
				URL:         baseURL,
				Timeout:     15,
				Show:        true,
				Wait:        "load",
				Render:      15000,
				OutputDir:   "./bil.html",
				UserDataDir: "./user_data",
				CookieFile:  "C:\\Users\\Administrator\\Downloads\\fetch_content-V2\\cookie.json",
				FetcherPath: "C:\\Users\\Administrator\\Downloads\\fetch_content-V2\\output\\fetch_content\\fetch_content.exe", // 指定 fetcher.exe 的路径
			}

			// 调用 FetchHTML 函数获取 HTML 内容
			htmlContent, err := parser.FetchHTML(options)
			if err != nil {
				resultsChan <- FetchResult{URL: baseURL, Data: nil, Error: err}
				return
			}

			// 解析 HTML 内容
			doc, err := parser.ParseHTMLs(htmlContent)
			if err != nil {
				resultsChan <- FetchResult{URL: baseURL, Data: nil, Error: err}
				return
			}

			// 查找所有 cardBox 节点
			cardBoxes, err := parser.FindNodes(doc, cardBoxXPath)
			if err != nil {
				resultsChan <- FetchResult{URL: baseURL, Data: nil, Error: err}
				return
			}

			var dataList []ExtractedData
			for _, cardBox := range cardBoxes {
				// 提取 <a class='cover'> 节点并获取 href 属性
				aNode, err := parser.FindNode(cardBox, hrefXPath)
				if err != nil || aNode == nil {
					log.Printf("未找到 a.cover 节点: %v", err)
					continue
				}
				href, err := parser.GetAttribute(aNode, "href")
				if err != nil {
					log.Printf("未能提取 href 属性: %v", err)
					href = ""
				} else {
					href = strings.Replace(href, "//", "", 1)
					href = addDomain(baseURL, href) // 处理相对路径
				}

				// 提取 <source type='image/webp'> 节点并获取 srcset 属性
				sourceNode, err := parser.FindNode(cardBox, imgXPath)
				if err != nil || sourceNode == nil {
					log.Printf("未找到 source[type='image/webp'] 节点: %v", err)
					continue
				}
				imgSrc, err := parser.GetAttribute(sourceNode, "srcset")
				if err != nil {
					log.Printf("未能提取 srcset 属性: %v", err)
					imgSrc = ""
				} else {
					// 处理可能的相对路径
					if strings.HasPrefix(imgSrc, "//") {
						imgSrc = "https:" + imgSrc
					} else if strings.HasPrefix(imgSrc, "/") {
						imgSrc = addDomain(baseURL, imgSrc)
					}
				}

				// 提取 title
				title, err := parser.GetText(cardBox, titleXPath)
				if err != nil {
					log.Printf("未能提取 title: %v", err)
					title = ""
				} else {
					title = strings.TrimSpace(title)
				}

				// 提取 time
				time, err := parser.GetText(cardBox, timeXPath)
				if err != nil {
					log.Printf("未能提取 time: %v", err)
					time = ""
				} else {
					time = strings.TrimSpace(time)
				}

				dataList = append(dataList, ExtractedData{
					Href:   href,
					ImgSrc: imgSrc,
					Title:  title,
					Time:   time,
				})
			}

			// 发送结果到通道
			resultsChan <- FetchResult{URL: baseURL, Data: dataList, Error: nil}
		}(baseURL)
	}

	// 等待所有 goroutine 完成
	wg.Wait()
	close(resultsChan)

	// 处理结果
	for result := range resultsChan {
		if result.Error != nil {
			log.Printf("URL: %s, 错误: %v\n", result.URL, result.Error)
			continue
		}
		fmt.Printf("URL: %s\n提取的内容:\n", result.URL)
		for _, item := range result.Data {
			fmt.Printf("Href: %s\nImgSrc: %s\nTitle: %s\nTime: %s\n--------\n", item.Href, item.ImgSrc, item.Title, item.Time)
		}
		fmt.Println("========")
	}
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
