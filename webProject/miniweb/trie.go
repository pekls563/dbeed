package miniweb

import "strings"

//前缀树实现动态路由,例如/hello/:name，可以匹配/hello/tt、hello/jack等,/hello/*name可以匹配/hello/a/v/c/d等
//动态路由不处理请求携带的参数,也就是?后的部分

//  /p/:lang/doc

type node struct {
	pattern  string  // 待匹配路由，例如 /p/:lang  //理解为根节点到该节点的路径（就是总的访问路径）
	part     string  // 路由中的一部分，例如 :lang  //理解为该节点包含的路径
	children []*node // 子节点，例如 [doc, tutorial, intro]
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

	//第一次调用,height=0,为根节点的深度
	if len(parts) == height {
		n.pattern = pattern
		return
	}

	//  pattern="/hello/*name/a/b/c"  parts=["hello","*name"]

	//  pattern="/hello/name/a/b/c"   parts=["hello","name","a","b","c"]

	//以 pattern="/hello/*name/a/b/c"  parts=["hello","*name"]为例
	//一共调用3次insert,根节点.part="",子节点.part="hello",孙子节点.part="*name",孙子节点.pattern=/hello/*name,再往下没有节点

	part := parts[height]
	//第一次part="hello",第二次part="*name"

	//若part="hello"未匹配成功，则为n节点新建孩子节点"hello"
	child := n.matchChild(part)
	if child == nil {
		child = &node{part: part, isWild: part[0] == ':' || part[0] == '*'}
		n.children = append(n.children, child)
	}
	//将height+1,调整到孩子节点的深度，child.insert()递归调用函数，为孩子添加孙子节点
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

	//最终返回对应pattern的最底层的孩子节点,即该子节点.pattern=/hello/*name
	//最终如果多个孩子节点能匹配到，返回最先匹配到的那个孩子节点
	for _, child := range children {
		result := child.search(parts, height+1)
		if result != nil {
			return result
		}
	}

	return nil
}
