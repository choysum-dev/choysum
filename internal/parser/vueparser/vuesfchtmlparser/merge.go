// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vuesfchtmlparser

import (
	"github.com/antchfx/htmlquery"
	xfmt "golang.org/x/exp/errors/fmt"
	"golang.org/x/net/html"
)

func CloneNode(n *html.Node) *html.Node {
	if n == nil {
		return nil
	}

	clone := &html.Node{
		Type:      n.Type,
		DataAtom:  n.DataAtom,
		Data:      n.Data,
		Namespace: n.Namespace,
	}

	if len(n.Attr) > 0 {
		clone.Attr = make([]html.Attribute, len(n.Attr))
		copy(clone.Attr, n.Attr)
	}

	// Keep references to all child nodes.
	var children []*html.Node
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		clonedChild := CloneNode(child)
		children = append(children, clonedChild)
	}

	// Rebuild the node relationships.
	for i, child := range children {
		child.Parent = clone
		if i == 0 {
			clone.FirstChild = child
		} else {
			children[i-1].NextSibling = child
			child.PrevSibling = children[i-1]
		}
	}

	return clone
}

func ApplyXPathToTemplate(sourceNode *html.Node, templateXpathNode *html.Node) (*html.Node, error) {
	if templateXpathNode == nil {
		return CloneNode(sourceNode), nil
	}

	mergedNode := CloneNode(sourceNode)

	// Query all xpath nodes in the template with case-insensitive matching
	// Using XPath translate() function to match both 'xpath' and 'Xpath'
	// translate and local-name() are XPath functions, see: https://github.com/antchfx/xpath/blob/master/README.md
	xpathNodes, err := htmlquery.QueryAll(templateXpathNode,
		"//*[translate(local-name(), 'XPATH', 'xpath')='xpath']")
	if err != nil {
		return nil, xfmt.Errorf("failed to query xpath nodes: %w", err)
	}

	// Process each xpath node
	for _, xpathNode := range xpathNodes {
		var expr, position string
		// Extract expr and position attributes
		for _, attr := range xpathNode.Attr {
			if attr.Key == "expr" {
				expr = attr.Val
			}
			if attr.Key == "position" {
				position = attr.Val
			}
		}
		if expr == "" {
			continue
		}
		if position == "" {
			position = "after"
		}

		destNodes, err := htmlquery.QueryAll(mergedNode, expr)
		if err != nil {
			return nil, xfmt.Errorf("failed to query node: %w", err)
		}
		if len(destNodes) == 0 {
			return nil, xfmt.Errorf("no node found for expr: %s", expr)
		}
		for _, destNode := range destNodes {
			if err := applyXPathNode(destNode, xpathNode, position); err != nil {
				return nil, err
			}
		}
	}

	return mergedNode, nil
}

func applyXPathNode(destNode *html.Node, xpathNode *html.Node, position string) error {
	switch position {
	case "before":
		return applyBefore(destNode, xpathNode)
	case "after":
		return applyAfter(destNode, xpathNode)
	case "inside":
		return applyInside(destNode, xpathNode)
	case "replace":
		return applyReplace(destNode, xpathNode)
	case "attribute":
		return applyAttributes(destNode, xpathNode)
	default:
		return xfmt.Errorf("unsupported position: %s", position)
	}
}

func applyBefore(destNode *html.Node, xpathNode *html.Node) error {
	parent := destNode.Parent
	if parent == nil {
		return xfmt.Errorf("dest node has no parent")
	}
	for child := xpathNode.FirstChild; child != nil; child = child.NextSibling {
		clonedNode := CloneNode(child)
		parent.InsertBefore(clonedNode, destNode)
	}
	return nil
}

func applyAfter(destNode *html.Node, xpathNode *html.Node) error {
	parent := destNode.Parent
	if parent == nil {
		return xfmt.Errorf("dest node has no parent")
	}

	// Record the next sibling of the target node as the insertion point.
	insertPoint := destNode.NextSibling

	// Collect and clone all nodes to insert first.
	var nodesToInsert []*html.Node
	for child := xpathNode.FirstChild; child != nil; child = child.NextSibling {
		clonedNode := CloneNode(child)
		clonedNode.Parent = parent
		nodesToInsert = append(nodesToInsert, clonedNode)
	}

	// Link the inserted nodes together.
	for i := 0; i < len(nodesToInsert); i++ {
		if i > 0 {
			nodesToInsert[i].PrevSibling = nodesToInsert[i-1]
		}
		if i < len(nodesToInsert)-1 {
			nodesToInsert[i].NextSibling = nodesToInsert[i+1]
		}
	}

	// Insert the nodes into the DOM tree.
	if len(nodesToInsert) > 0 {
		// Connect the target node with the inserted nodes.
		nodesToInsert[0].PrevSibling = destNode
		if insertPoint != nil {
			nodesToInsert[len(nodesToInsert)-1].NextSibling = insertPoint
			insertPoint.PrevSibling = nodesToInsert[len(nodesToInsert)-1]
		}
		destNode.NextSibling = nodesToInsert[0]

		// Update the parent LastChild when this becomes the final node.
		if insertPoint == nil {
			parent.LastChild = nodesToInsert[len(nodesToInsert)-1]
		}
	}

	return nil
}

func applyInside(destNode *html.Node, xpathNode *html.Node) error {
	for child := xpathNode.FirstChild; child != nil; child = child.NextSibling {
		clonedNode := CloneNode(child)
		if destNode.FirstChild == nil {
			destNode.FirstChild = clonedNode
		} else {
			lastChild := destNode.FirstChild
			for lastChild.NextSibling != nil {
				lastChild = lastChild.NextSibling
			}
			lastChild.NextSibling = clonedNode
		}
		clonedNode.Parent = destNode
	}
	return nil
}

func applyReplace(destNode *html.Node, xpathNode *html.Node) error {
	parent := destNode.Parent
	if parent == nil {
		return xfmt.Errorf("dest node has no parent")
	}

	var firstNode, prevNode *html.Node
	for child := xpathNode.FirstChild; child != nil; child = child.NextSibling {
		clonedNode := CloneNode(child)
		if firstNode == nil {
			firstNode = clonedNode
			parent.InsertBefore(clonedNode, destNode)
		} else {
			parent.InsertBefore(clonedNode, prevNode.NextSibling)
		}
		prevNode = clonedNode
	}
	parent.RemoveChild(destNode)
	return nil
}

func applyAttributes(destNode *html.Node, xpathNode *html.Node) error {
	// Get attrName and attrValue from xpath node attributes
	// Support both camelCase and kebab-case formats
	var attrName, attrValue string
	for _, attr := range xpathNode.Attr {
		switch attr.Key {
		case "attrName", "attr-name":
			attrName = attr.Val
		case "attrValue", "attr-value":
			attrValue = attr.Val
		}
	}

	// Validate required attributes
	if attrName == "" {
		return xfmt.Errorf("attrName/attr-name is required for attribute position")
	}
	if attrValue == "" {
		return xfmt.Errorf("attrValue/attr-value is required for attribute position")
	}

	// Update or add the attribute to destination node
	found := false
	for i, attr := range destNode.Attr {
		if attr.Key == attrName {
			destNode.Attr[i].Val = attrValue
			found = true
			break
		}
	}

	if !found {
		destNode.Attr = append(destNode.Attr, html.Attribute{
			Key: attrName,
			Val: attrValue,
		})
	}

	return nil
}
