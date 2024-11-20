package parser

import (
	"fmt"
	"strings"

	"github.com/antchfx/htmlquery"
	"golang.org/x/net/html"
)

// ParseHTMLs 解析 HTML 内容并返回文档节点
func ParseHTMLs(htmlContent string) (*html.Node, error) {
	doc, err := htmlquery.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil, fmt.Errorf("解析 HTML 失败: %v", err)
	}
	return doc, nil
}

// FindNodes 根据 XPath 表达式查找节点
func FindNodes(doc *html.Node, xpathExpr string) ([]*html.Node, error) {
	nodes := htmlquery.Find(doc, xpathExpr)
	if len(nodes) == 0 {
		return nil, fmt.Errorf("未找到任何匹配的内容，XPath: %s", xpathExpr)
	}
	return nodes, nil
}

// FindNode 根据 XPath 表达式查找第一个匹配的节点
func FindNode(node *html.Node, xpathExpr string) (*html.Node, error) {
	foundNode := htmlquery.FindOne(node, xpathExpr)
	if foundNode == nil {
		return nil, fmt.Errorf("未找到匹配的节点，XPath: %s", xpathExpr)
	}
	return foundNode, nil
}

// GetAttribute 获取节点的指定属性值
func GetAttribute(node *html.Node, attr string) (string, error) {
	value := htmlquery.SelectAttr(node, attr)
	if value == "" {
		return "", fmt.Errorf("属性 '%s' 不存在或为空", attr)
	}
	return value, nil
}

// GetText 获取节点的文本内容
func GetText(node *html.Node, xpathExpr string) (string, error) {
	targetNode := htmlquery.FindOne(node, xpathExpr)
	if targetNode != nil {
		return strings.TrimSpace(htmlquery.InnerText(targetNode)), nil
	}
	return "", fmt.Errorf("未找到匹配的文本节点，XPath: %s", xpathExpr)
}

// RenderHTML 用于调试，返回节点的 HTML 字符串表示
func RenderHTML(node *html.Node) string {
	return htmlquery.OutputHTML(node, true)
}
