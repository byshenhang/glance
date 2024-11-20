package parser

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// FetchOptions 定义调用 fetcher.exe 所需的参数
type FetchOptions struct {
	URL         string
	Timeout     int
	Show        bool
	Wait        string
	Render      int
	UserDataDir string
	OutputDir   string
	CookieFile  string
	FetcherPath string // 新增字段，用于指定 fetcher.exe 的路径
}

// FetchHTML 调用 fetcher.exe 并返回抓取到的 HTML 内容，支持上下文超时
func FetchHTML(options FetchOptions) (string, error) {
	// 构建命令参数
	args := []string{
		options.URL,
		"--timeout", fmt.Sprintf("%d", options.Timeout),
		"--wait", options.Wait,
		"--render", fmt.Sprintf("%d", options.Render),
		"--user-data-dir", options.UserDataDir,
		"--output", options.OutputDir,
	}

	if options.Show {
		args = append(args, "--show")
	}
	if options.CookieFile != "" {
		args = append(args, "--cookie-file", options.CookieFile)
	}

	// 确定 fetcher.exe 的路径
	fetcherPath := "fetcher.exe" // 默认在 PATH 中
	if options.FetcherPath != "" {
		fetcherPath = options.FetcherPath
	}

	// 创建带有超时的上下文
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(options.Timeout+5)*time.Second)
	defer cancel()

	// 创建命令
	cmd := exec.CommandContext(ctx, fetcherPath, args...)

	// 捕获 stdout 和 stderr
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 运行命令
	err := cmd.Run()
	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("执行 fetcher.exe 超时")
	}
	if err != nil {
		return "", fmt.Errorf("执行 fetcher.exe 失败: %v, Stderr: %s", err, stderr.String())
	}

	// 获取文件路径
	filePath := strings.TrimSpace(stdout.String())
	if filePath == "" {
		return "", errors.New("fetcher.exe 未返回文件路径")
	}

	// 读取文件内容
	htmlContent, err := os.ReadFile(options.OutputDir)
	if err != nil {
		return "", fmt.Errorf("读取文件失败: %v", err)
	}

	return string(htmlContent), nil
}
