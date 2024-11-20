package tool

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	appID  = "20200208000381654"    // 固定 AppID
	appKey = "FPTQJyuTh3BBvcbT8cby" // 固定 AppKey
	apiURL = "http://api.fanyi.baidu.com/api/trans/vip/translate"
)

var (
	cache = make(map[string]string)
	mu    sync.Mutex
)

type translateResponse struct {
	TransResult []struct {
		Dst string `json:"dst"`
	} `json:"trans_result"`
	ErrorMsg string `json:"error_msg,omitempty"`
}

// 对外暴露的翻译函数，设置默认语言
func TranslateWithDefaults(query string) (string, error) {
	return Translate(query, "en", "zh")
}

// Translate translates the given text. Defaults to "en" (English) to "zh" (Simplified Chinese) if not specified.
func Translate(query, fromLang, toLang string) (string, error) {
	mu.Lock()
	if result, found := cache[query]; found {
		mu.Unlock()
		return result, nil
	}
	mu.Unlock()

	// Generate salt and sign
	rand.Seed(time.Now().UnixNano())
	salt := rand.Intn(65536-32768) + 32768
	sign := generateSign(query, salt)

	// Build request
	data := url.Values{}
	data.Set("appid", appID)
	data.Set("q", query)
	data.Set("from", fromLang)
	data.Set("to", toLang)
	data.Set("salt", fmt.Sprintf("%d", salt))
	data.Set("sign", sign)

	resp, err := http.PostForm(apiURL, data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result translateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if len(result.TransResult) > 0 {
		finalResult := ""
		for _, res := range result.TransResult {
			finalResult += res.Dst
		}

		mu.Lock()
		cache[query] = finalResult
		mu.Unlock()

		return finalResult, nil
	}

	return "", fmt.Errorf("translation failed: %s", result.ErrorMsg)
}

// generateSign generates an MD5 hash for the translation request
func generateSign(query string, salt int) string {
	signStr := fmt.Sprintf("%s%s%d%s", appID, query, salt, appKey)
	hash := md5.Sum([]byte(signStr))
	return hex.EncodeToString(hash[:])
}
