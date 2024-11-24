package tool

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"
)

// DynamicKeyHelper 是一个用于生成和验证动态密钥的助手结构体。
type DynamicKeyHelper struct{}

// NewDynamicKeyHelper 初始化一个新的 DynamicKeyHelper。
// 由于不再使用缓存，该函数可以返回一个空的 DynamicKeyHelper。
func NewDynamicKeyHelper() *DynamicKeyHelper {
	return &DynamicKeyHelper{}
}

// GenerateKey 基于当前时间和时间步长生成一个动态密钥。
// secret 是共享密钥，timeStep 是时间步长（秒）。
func (d *DynamicKeyHelper) GenerateKey(secret string, timeStep int) (string, error) {
	currentTime := time.Now().Unix()
	timeInterval := currentTime / int64(timeStep)

	message := []byte(fmt.Sprintf("%d", timeInterval))
	secretBytes := []byte(secret)

	h := hmac.New(sha256.New, secretBytes)
	_, err := h.Write(message)
	if err != nil {
		return "", err
	}
	hmacDigest := h.Sum(nil)

	dynamicKey := base64.URLEncoding.EncodeToString(hmacDigest)
	return dynamicKey, nil
}

// VerifyKey 验证接收到的动态密钥是否有效。
// secret 是共享密钥，receivedKey 是接收到的动态密钥。
// timeStep 是时间步长（秒），tolerance 是允许的时间窗口偏移量。
// 注意：由于已移除缓存，无法防止密钥的重放攻击。
func (d *DynamicKeyHelper) VerifyKey(secret string, receivedKey string, timeStep int, tolerance int) (bool, error) {
	secretBytes := []byte(secret)
	currentTime := time.Now().Unix()
	currentInterval := currentTime / int64(timeStep)

	for offset := -tolerance; offset <= tolerance; offset++ {
		timeInterval := currentInterval + int64(offset)
		message := []byte(fmt.Sprintf("%d", timeInterval))

		h := hmac.New(sha256.New, secretBytes)
		_, err := h.Write(message)
		if err != nil {
			return false, err
		}
		expectedHmac := h.Sum(nil)
		expectedKey := base64.URLEncoding.EncodeToString(expectedHmac)

		if hmac.Equal([]byte(receivedKey), []byte(expectedKey)) {
			return true, nil
		}
	}
	return false, fmt.Errorf("密钥验证失败")
}
