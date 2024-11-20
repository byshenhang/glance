package tool

import (
	"context"
	"fmt"
	"sync"

	openai "github.com/sashabaranov/go-openai"
)

var (
	client     *openai.Client
	clientOnce sync.Once
	initError  error
)

// 内置的 API 令牌和自定义域名
const (
	apiToken     = "sk-ia3AsF58W0KL3xlI51774aBe9d344118896393D6F4F55f6b" // 请替换为您的实际 API 令牌
	customDomain = "https://api.wssh.one/v1"                             // 请替换为您的自定义域名
	model        = "SparkDesk-v4.0"                                      // 使用的模型名称
)

// initializeClient 初始化 OpenAI 客户端（单例）
func initializeClient() error {
	clientOnce.Do(func() {
		config := openai.DefaultConfig(apiToken)
		config.BaseURL = customDomain
		client = openai.NewClientWithConfig(config)

	})
	return initError
}

// PromptSummarize 接受内容字符串并返回中文总结
func PromptSummarize(content string) (string, error) {
	// 确保客户端已初始化
	if err := initializeClient(); err != nil {
		return "", err
	}

	// 定义对话消息，包括系统角色和用户输入
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "你是一个有用的助手，负责总结文章内容。请用中文简明扼要地总结以下内容，不需要任何额外的解释或说明。\\n\\n 如下是需要总结的内容 \\",
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: content,
		},
	}

	// 创建 ChatCompletion 请求
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    model,
			Messages: messages,
		},
	)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	// 返回纯中文总结
	return resp.Choices[0].Message.Content, nil
}

// PromptTranslate 接受内容字符串并返回中文翻译
func PromptTranslate(content string) (string, error) {
	// 确保客户端已初始化
	if err := initializeClient(); err != nil {
		return "", err
	}

	// 定义对话消息，包括系统角色和用户输入
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: "你是一名专业的翻译员，负责将英文文本翻译成中文。请仅提供翻译后的中文内容，不需要任何额外的解释或说明。\n\n 如下是翻译内容 \" ",
		},
		{
			Role:    openai.ChatMessageRoleUser,
			Content: content,
		},
	}

	// 创建 ChatCompletion 请求
	resp, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model:    model,
			Messages: messages,
		},
	)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no response from OpenAI")
	}

	// 返回纯中文翻译
	return resp.Choices[0].Message.Content, nil
}
