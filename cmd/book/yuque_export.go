package book

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

/**********************
 *
 *   数据结构
 *
 **********************/

type CatalogResp struct {
	Data []CatalogNode `json:"data"`
}

type CatalogNode struct {
	Type       string `json:"type"`
	Title      string `json:"title"`
	UUID       string `json:"uuid"`
	URL        string `json:"url"`
	ParentUUID string `json:"parent_uuid"`
}

type TreeNode struct {
	Node     CatalogNode
	Children []*TreeNode
}

/**********************
 *
 *   限速器（串行）
 *
 **********************/

type Limiter struct {
	interval time.Duration
}

func NewLimiter(interval time.Duration) *Limiter {
	return &Limiter{interval: interval}
}

func (l *Limiter) Wait() {
	jitter := time.Duration(rand.Int63n(int64(l.interval / 2)))
	time.Sleep(l.interval + jitter)
}

/**********************
 *
 *   HTTP GET + 重试
 *
 **********************/

func httpGetRetry(url, cookie string, limiter *Limiter) ([]byte, error) {
	var lastErr error

	for i := 0; i < 3; i++ {
		limiter.Wait()

		req, _ := http.NewRequest("GET", url, nil)
		req.Header.Set("User-Agent", "Mozilla/5.0")
		req.Header.Set("Accept", "*/*")
		if cookie != "" {
			req.Header.Set("Cookie", cookie)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == 200 {
			return io.ReadAll(resp.Body)
		}

		lastErr = fmt.Errorf("status=%d", resp.StatusCode)
		time.Sleep(2 * time.Second)
	}
	return nil, lastErr
}

/**********************
 *
 *   构建树
 *
 **********************/

func BuildTree(nodes []CatalogNode) []*TreeNode {
	nodeMap := make(map[string]*TreeNode)

	for _, n := range nodes {
		nodeMap[n.UUID] = &TreeNode{Node: n}
	}

	var roots []*TreeNode

	for _, n := range nodes {
		cur := nodeMap[n.UUID]

		if n.ParentUUID == "" {
			roots = append(roots, cur)
			continue
		}

		if parent, ok := nodeMap[n.ParentUUID]; ok {
			parent.Children = append(parent.Children, cur)
		} else {
			roots = append(roots, cur)
		}
	}

	return roots
}

/**********************
 *
 *   下载 Markdown
 *
 **********************/

func downloadMarkdown(base, slug, cookie string, limiter *Limiter) (string, error) {
	url := fmt.Sprintf("%s/%s/markdown?attachment=true&latexcode=false&anchor=false&linebreak=false", base, slug)

	body, err := httpGetRetry(url, cookie, limiter)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func saveNode(node *TreeNode, base, cookie, dir string, limiter *Limiter) {
	// 1) 目录节点：先创建目录（仅 TITLE 创建目录；其他类型沿用当前 dir）
	curDir := dir
	if node.Node.Type == "TITLE" {
		title := safeName(node.Node.Title, "UNTITLED")
		curDir = filepath.Join(dir, title)
		_ = os.MkdirAll(curDir, 0755)
	}

	// 2) 文档节点：下载并保存（文件名保证唯一）
	if node.Node.Type == "DOC" && node.Node.URL != "" {
		fmt.Println("visit doc:", node.Node.Title, "slug:", node.Node.URL, "dir:", curDir)

		file := uniqueMarkdownPath(curDir, node.Node.Title, node.Node.URL, node.Node.UUID)

		// 断点续传：存在且 size > 0 才跳过
		if fi, err := os.Stat(file); err == nil && fi.Size() > 0 {
			fmt.Println("skip:", file)
		} else {
			md, err := downloadMarkdown(base, node.Node.URL, cookie, limiter)
			if err != nil {
				fmt.Println("fail:", node.Node.Title, "slug:", node.Node.URL, "err:", err)
			} else {
				_ = os.WriteFile(file, []byte(md), 0644)
				fmt.Println("saved:", file)
			}
		}
	}

	// 3) 无论什么类型，都递归 children，确保遍历完整
	for _, c := range node.Children {
		saveNode(c, base, cookie, curDir, limiter)
	}
}

// title 作为目录/文件名的安全处理，title 为空给 fallback
func safeName(title, fallback string) string {
	s := sanitize(title)
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	return s
}

// 文件名唯一：title + slug(or uuid) 兜底
func uniqueMarkdownPath(dir, title, slug, uuid string) string {
	name := safeName(title, "UNTITLED")

	// 推荐：标题 + slug，稳定且可读
	// slug 可能含特殊字符，这里再 sanitize 一次
	suffix := sanitize(slug)
	suffix = strings.TrimSpace(suffix)
	if suffix == "" {
		suffix = sanitize(uuid)
	}
	if suffix != "" {
		name = fmt.Sprintf("%s__%s", name, suffix)
	}

	return filepath.Join(dir, name+".md")
}

/**********************
 *
 *   导出
 *
 **********************/

func Export(bookID string, namespace, cookie, output string) error {
	api := fmt.Sprintf("https://www.yuque.com/api/catalog_nodes?book_id=%s", bookID)

	limiter := NewLimiter(3 * time.Second) // 每 3~4.5 秒一次

	body, err := httpGetRetry(api, cookie, limiter)
	if err != nil {
		return err
	}

	var resp CatalogResp
	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}

	tree := BuildTree(resp.Data)

	base := fmt.Sprintf("https://www.yuque.com/%s", namespace)

	for _, root := range tree {
		saveNode(root, base, cookie, output, limiter)
	}

	return nil
}

/**********************
 *
 *   工具
 *
 **********************/

func sanitize(name string) string {
	re := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)
	return re.ReplaceAllString(name, "_")
}
