package miniweb

import "strings"

type node struct {
	pattern  string  // 仅仅在叶子节点中有值，表示构建前缀树时插入的某个字符串
	part     string  // 路由中的一部分，例如 :lang  //理解为该节点包含的路径
	children []*node // 子节点，
	isWild   bool    // 是否精确匹配，part 含有 : 或 * 时为true
}

// 第一个匹配成功的节点，用于插入,返回n的所有子节点中第一次匹配成功的节点
func (n *node) matchChild(part string) *node {
	for _, child := range n.children {
		if child.part == part || child.isWild {
			//part相等或者part含有:或*时都算匹配成功
			return child
		}
	}
	return nil
}

// 所有匹配成功的节点，用于查找,返回n的所有子节点中所有匹配成功的节点
func (n *node) matchChildren(part string) []*node {
	nodes := make([]*node, 0)
	for _, child := range n.children {
		if child.part == part || child.isWild {
			nodes = append(nodes, child)
		}
	}
	return nodes
}

//支持前缀树节点的插入和查询
func (n *node) insert(pattern string, parts []string, height int) {

	if len(parts) == height {
		n.pattern = pattern
		return
	}
	part := parts[height]
	child := n.matchChild(part)
	if child == nil {
		child = &node{part: part, isWild: part[0] == ':' || part[0] == '*'}
		n.children = append(n.children, child)
	}
	child.insert(pattern, parts, height+1)
}

func (n *node) search(parts []string, height int) *node {
	//parts=["hello","a"]
	if len(parts) == height || strings.HasPrefix(n.part, "*") {
		if n.pattern == "" {
			return nil
		}
		return n
	}

	//part="hello"
	part := parts[height]

	//matchChildren已经限制了子节点part部分相等,if len(parts) == height || strings.HasPrefix(n.part, "*") 可以不判断part相等
	children := n.matchChildren(part)

	//最终如果多个孩子节点能匹配到，返回最先匹配到的那个孩子节点
	//深度优先搜索
	for _, child := range children {
		result := child.search(parts, height+1)
		if result != nil {
			return result
		}
	}

	return nil
}
