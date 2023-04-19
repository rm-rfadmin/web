package main

import (
	"fmt"
	"net/http"
	"strings"
)

// node 结构体标识路由树的节点
type node struct {
	pattern  string  // 路由规则
	part     string  // 路由规则中的一个部分
	children []*node // 子节点
	isWild   bool    // 是否为通配符
}

// matchChild 方法用于在子节点中查找指定 part 匹配的节点
func (n *node) matchChild(part string) *node {
	for _, child := range n.children {
		if child.part == part || child.isWild {
			return child
		}
	}
	return nil
}

// insert 方法用于向路由树中插入新的节点，并递归调用自身完成整个节点的插入过程
func (n *node) insert(pattern string, parts []string, height int) {
	// 如果当前已经到达最后一层，即parts 数组为空，则将节点的 pattern 字段设置为当前路由规则，
	// 兵返回结束递归
	if len(parts) == height {
		n.pattern = pattern
		return
	}

	// 否则，取出 parts 数组中当前层对应的部分 part， 并在当前节点的子节点中查找是否含有匹配的节点
	part := parts[height]
	child := n.matchChild(part)

	// 如果没有匹配的节点，则创建一个新节点，并将其添加到当前节点的子节点中
	if child == nil {
		child = &node{part: part, isWild: part[0] == ':'}
		n.children = append(n.children, child)
	}

	// 递归调用 insert 方法，将当前节点设置为子节点，高度加 1，继续向下一层递归
	child.insert(pattern, parts, height+1)
}

// search 方法用于查找路由树中是否存在匹配的路由规则
func (n *node) search(parts []string, height int) *node {
	// 如果当前已经到达最后一层，即parts 数组为空，则判断当前节点的 pattern 字段是否为空，
	if len(parts) == height || strings.HasPrefix(n.part, "*") {
		if n.pattern == "" {
			return nil
		}
		return n
	}

	// 否则，取出 parts 数组中当前层对应的部分 part， 并在当前节点的子节点中查找是否含有匹配的节点
	part := parts[height]
	child := n.matchChild(part)

	// 如果没有匹配的节点，则返回 nil
	if child == nil {
		return nil
	}

	// 递归调用 search 方法，将当前节点设置为子节点，高度加 1，继续向下一层递归
	return child.search(parts, height+1)
}

// router 结构体用于实现路由树的插入、查找和路由处理
type router struct {
	roots    map[string]*node            // 用于存储不同 HTTP 方法对应的路由树的根节点
	handlers map[string]http.HandlerFunc // 用于存储路由规则和对应的处理函数
}

// newRouter 方法用于创建一个路由树
func newRouter() *router {
	return &router{
		roots:    make(map[string]*node),            // 初始化 roots 字段 存储不同 HTTP 方法对应的路由树的根节点
		handlers: make(map[string]http.HandlerFunc), // 初始化 handlers 字段 用于存储路由规则和对应的处理函数
	}
}

// parsePattern 方法用于解析路由规则，将路由规则按照 / 分割，将分割后的结果存储到切片中
func parsePattern(pattern string) []string {
	parts := strings.Split(pattern, "/")
	result := make([]string, 0)
	for _, part := range parts {
		if part != "" {
			result = append(result, part)
			if part[0] == '*' {
				break
			}
		}
	}
	return result
}

func (r *router) addRoute(method, pattern string, handler http.HandlerFunc) {
	parts := parsePattern(pattern)

	key := method + "-" + pattern
	_, ok := r.roots[method]
	if !ok {
		r.roots[method] = &node{}
	}
	r.roots[method].insert(pattern, parts, 0)
	r.handlers[key] = handler
}

func (r *router) getRoute(method, path string) (*node, map[string]string) {
	searchParts := parsePattern(path)
	params := make(map[string]string)

	root, ok := r.roots[method]
	if !ok {
		return nil, nil
	}

	n := root.search(searchParts, 0)
	if n == nil {
		return nil, nil
	}

	parts := parsePattern(n.pattern)
	for i, part := range parts {
		if part[0] == ':' {
			params[part[1:]] = searchParts[i]
		}
		if part[0] == '*' && len(part) > 1 {
			params[part[1:]] = strings.Join(searchParts[i:], "/")
			break
		}
	}

	return n, params
}

func (r *router) handle(c http.ResponseWriter, req *http.Request) {
	key := req.Method + "-" + req.URL.Path

	handler, ok := r.handlers[key]
	if !ok {
		http.NotFound(c, req)
		return
	}

	handler(c, req)
}

func main() {
	r := newRouter()
	r.addRoute("GET", "/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "Hello, World!")
	})
	r.addRoute("GET", "/hello/:name", func(w http.ResponseWriter, r *http.Request) {
		params := r.Context().Value("params").(map[string]string)
		fmt.Fprintf(w, "Hello, %s!", params["name"])
	})
	r.addRoute("GET", "/user/*action", func(w http.ResponseWriter, r *http.Request) {
		params := r.Context().Value("params").(map[string]string)
		fmt.Fprintf(w, "Action: %s", params["action"])
	})
}
