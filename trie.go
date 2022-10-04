package gig

import (
	"fmt"
	"strings"
)

type node struct {
	pattern  string  // 待匹配路由，驶入 /p/:lang
	part     string  // 路由中的一部分，例如 :lang
	children []*node // 子节点， 例如[doc, toturial, intro]
	isWild   bool    // 是否精确匹配，part含有 ：或 * 时为true
}

// String方法
func (n *node) String() string {
	return fmt.Sprintf("node{pattern=%s, part=%s, isWild=%t}", n.pattern, n.part, n.isWild)
}

// 插入新的节点
// 递归查找每一层的节点，如果没有匹配到当前 part 的节点，则新建一个
// 有一点需要注意，/p/:lang/doc 只有在第三层节点，即 doc 节点，pattern 才会设置为 /p/:lang/doc
func (n *node) insert(pattern string, parts []string, height int) {
	// 每递归一次深度+1，如果节点数等于递归深度则退出
	if len(parts) == height {
		n.pattern = pattern
		return
	}
	part := parts[height]
	child := n.matchChild(part)
	if child == nil {
		// 新建前缀树节点
		child = &node{
			part:   part,
			isWild: part[0] == ':' || part[0] == '*',
		}
		n.children = append(n.children, child)
	}
	child.insert(pattern, parts, height+1)
}

// 查找节点
// 递归查询每一层的节点，退出规则是匹配到了 * 或匹配失败 或匹配到了 len(parts) 层节点
func (n *node) search(parts []string, height int) *node {
	if len(parts) == height || strings.HasPrefix(n.part, "*") {
		if n.pattern == "" {
			return nil
		}
		return n
	}

	part := parts[height]
	children := n.matchChildren(part)

	for _, child := range children {
		result := child.search(parts, height+1)
		if result != nil {
			return result
		}
	}

	return nil
}

// 遍历节点
func (n *node) travel(list *[]*node) {
	if n.pattern != "" {
		*list = append(*list, n)
	}
	for _, child := range n.children {
		child.travel(list)
	}
}

// 第一个匹配成功的节点，用于插入
func (n *node) matchChild(part string) *node {
	for _, child := range n.children {
		if child.part == part || child.isWild {
			return child
		}
	}
	return nil
}

// 所有匹配成功的节点，用于查找
func (n *node) matchChildren(part string) []*node {
	nodes := make([]*node, 0)
	for _, child := range n.children {
		if child.part == part || child.isWild {
			nodes = append(nodes, child)
		}
	}
	return nodes
}
