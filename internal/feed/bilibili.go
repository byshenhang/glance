// internal/feed/bilibili.go
package feed

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// 定义 Bilibili API 响应的结构体
type bilibiliApiResponse struct {
	Code    int          `json:"code"`
	Message string       `json:"message"`
	Ttl     int          `json:"ttl"`
	Data    bilibiliData `json:"data"`
}

type bilibiliData struct {
	Note string          `json:"note"`
	List []bilibiliVideo `json:"list"`
}

type bilibiliVideo struct {
	Aid         int64             `json:"aid"`
	Videos      int               `json:"videos"`
	Tid         int               `json:"tid"`
	Tname       string            `json:"tname"`
	Copyright   int               `json:"copyright"`
	Pic         string            `json:"pic"`
	Title       string            `json:"title"`
	Pubdate     int64             `json:"pubdate"`
	Ctime       int64             `json:"ctime"`
	Desc        string            `json:"desc"`
	State       int               `json:"state"`
	Duration    int               `json:"duration"`
	Rights      bilibiliRights    `json:"rights"`
	Owner       bilibiliOwner     `json:"owner"`
	Stat        bilibiliStat      `json:"stat"`
	Dynamic     string            `json:"dynamic"`
	Cid         int64             `json:"cid"`
	Dimension   bilibiliDimension `json:"dimension"`
	ShortLinkV2 string            `json:"short_link_v2"`
	FirstFrame  string            `json:"first_frame"`
	PubLocation string            `json:"pub_location"`
	Cover43     string            `json:"cover43"`
	Bvid        string            `json:"bvid"`
	Score       int               `json:"score"`
	EnableVt    int               `json:"enable_vt"`
}

type bilibiliRights struct {
	Bp            int `json:"bp"`
	Elec          int `json:"elec"`
	Download      int `json:"download"`
	Movie         int `json:"movie"`
	Pay           int `json:"pay"`
	Hd5           int `json:"hd5"`
	NoReprint     int `json:"no_reprint"`
	Autoplay      int `json:"autoplay"`
	UgcPay        int `json:"ugc_pay"`
	IsCooperation int `json:"is_cooperation"`
	UgcPayPreview int `json:"ugc_pay_preview"`
	NoBackground  int `json:"no_background"`
	ArcPay        int `json:"arc_pay"`
	PayFreeWatch  int `json:"pay_free_watch"`
}

type bilibiliOwner struct {
	Mid  int64  `json:"mid"`
	Name string `json:"name"`
	Face string `json:"face"`
}

type bilibiliStat struct {
	Aid      int64 `json:"aid"`
	View     int   `json:"view"`
	Danmaku  int   `json:"danmaku"`
	Reply    int   `json:"reply"`
	Favorite int   `json:"favorite"`
	Coin     int   `json:"coin"`
	Share    int   `json:"share"`
	NowRank  int   `json:"now_rank"`
	HisRank  int   `json:"his_rank"`
	Like     int   `json:"like"`
	Dislike  int   `json:"dislike"`
	Vt       int   `json:"vt"`
	Vv       int   `json:"vv"`
}

type bilibiliDimension struct {
	Width  int `json:"width"`
	Height int `json:"height"`
	Rotate int `json:"rotate"`
}

// 将 Bilibili 的发布时间（Unix 时间戳）转换为 time.Time
func parseBilibiliPubDate(t int64) time.Time {
	return time.Unix(t, 0)
}

func FetchBilibiliChannelUploads(channelIds []string, videoUrlTemplate string, includeShorts bool) (Videos, error) {
	requests := make([]*http.Request, 0, 1)

	// 获取代理 IP
	proxyAPI := "http://v2.api.juliangip.com/company/postpay/getips?auto_white=1&num=1&pt=1&result_type=text&split=1&trade_no=6047294990559659&sign=c2e6d4a312bdd2dec56fe602e78b1004"
	proxyIP, err := getProxyIP(proxyAPI)
	if err != nil {
		slog.Error("Failed to get proxy IP", "error", err)
		return nil, fmt.Errorf("failed to get proxy IP: %v", err)
	}
	slog.Info("获取到的代理IP", "proxy", proxyIP)

	// 解析代理 URL
	proxyURL, err := url.Parse(fmt.Sprintf("http://%s", proxyIP))
	if err != nil {
		slog.Error("Invalid proxy URL", "proxy", proxyIP, "error", err)
		return nil, fmt.Errorf("invalid proxy URL: %v", err)
	}

	// 配置 Transport 使用代理
	transport := &http.Transport{
		Proxy: http.ProxyURL(proxyURL),
	}

	// 配置 HTTP 客户端
	bililiclient := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	// 构造 API URL，这里假设使用排名接口，不再逐个频道请求
	apiUrl := "https://api.bilibili.com/x/web-interface/ranking/v2"

	request, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		slog.Error("Failed to create request", "channel", "ranking", "error", err)
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	// 设置请求头
	headers := map[string]string{
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36",
		"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
		"Accept-Language": "en-US,en;q=0.5",
		"Connection":      "keep-alive",
	}

	// 遍历设置所有的头部
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	requests = append(requests, request)

	// 使用与 YouTube 相同的并发模型
	job := newJob(decodeJsonFromRequestTask[bilibiliApiResponse](bililiclient), requests).withWorkers(30)

	responses, errs, err := workerPoolDo(job)

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNoContent, err)
	}

	videos := make(Videos, 0, 10) // 仅获取前10个视频

	var failed int

	for i, response := range responses {
		if errs[i] != nil {
			failed++
			slog.Error("Failed to fetch Bilibili feed", "error", errs[i])
			continue
		}

		// 直接使用 response 作为 bilibiliApiResponse 类型，无需类型断言
		apiResp := response

		if apiResp.Code != 0 {
			failed++
			slog.Error("Bilibili API returned non-zero code", "code", apiResp.Code, "message", apiResp.Message)
			continue
		}

		// 仅获取前10个视频
		for idx, video := range apiResp.Data.List {
			if idx >= 10 {
				break
			}

			// 构造视频 URL
			var videoUrl string

			if videoUrlTemplate == "" {
				videoUrl = video.ShortLinkV2
			} else {
				videoUrl = strings.ReplaceAll(videoUrlTemplate, "{VIDEO-ID}", video.Bvid)
			}

			// 构造 Author URL，假设为 Bilibili 的个人空间
			authorUrl := fmt.Sprintf("https://space.bilibili.com/%d/video", video.Owner.Mid)

			videos = append(videos, Video{
				ThumbnailUrl: video.Pic,
				Title:        video.Title,
				Url:          videoUrl,
				Author:       video.Owner.Name,
				AuthorUrl:    authorUrl,
				TimePosted:   parseBilibiliPubDate(video.Pubdate),
			})
		}
	}

	if len(videos) == 0 {
		return nil, ErrNoContent
	}

	// 假设 Videos 类型有 SortByNewest 方法
	// videos.SortByNewest()

	if failed > 0 {
		return videos, fmt.Errorf("%w: missing videos from %d channels", ErrPartialContent, failed)
	}

	return videos, nil
}

// 获取代理 IP 的辅助函数
func getProxyIP(apiURL string) (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	proxyIP := strings.TrimSpace(string(body))
	if proxyIP == "" {
		return "", fmt.Errorf("empty proxy IP received")
	}

	return proxyIP, nil
}

// decodeJsonFromRequestTaskBillili 是一个通用的任务函数，用于解析 JSON 响应
func decodeJsonFromRequestTaskBillili[T any](client *http.Client) func(*http.Request) (any, error) {
	return func(req *http.Request) (any, error) {
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("non-200 status code: %d", resp.StatusCode)
		}

		var result T
		decoder := json.NewDecoder(resp.Body)
		err = decoder.Decode(&result)
		if err != nil {
			return nil, err
		}

		return result, nil
	}
}
