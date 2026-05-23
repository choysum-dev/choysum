// SPDX-FileCopyrightText: 2026-present Brian Wang <wangbuke@gmail.com>
// SPDX-License-Identifier: LGPL-3.0-or-later

package vuesfchtmlparser

import (
	"io"
	"strings"

	"github.com/antchfx/htmlquery"
	xfmt "golang.org/x/exp/errors/fmt"
	"golang.org/x/net/html"
)

func RenderVueSfcFromHtmlNode(node *html.Node) (string, error) {
	var buf strings.Builder
	for n := node.FirstChild; n != nil; n = n.NextSibling {
		err := renderNode(&buf, n)
		if err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

func renderNode(w io.Writer, n *html.Node) error {
	switch n.Type {
	case html.TextNode:
		if _, err := w.Write([]byte(n.Data)); err != nil {
			return err
		}
		return nil
	case html.ElementNode:
		if _, err := w.Write([]byte("<")); err != nil {
			return err
		}
		if _, err := w.Write([]byte(n.Data)); err != nil {
			return err
		}

		// Render attributes without escaping values
		for _, attr := range n.Attr {
			if _, err := w.Write([]byte(" ")); err != nil {
				return err
			}
			if _, err := w.Write([]byte(attr.Key)); err != nil {
				return err
			}
			if _, err := w.Write([]byte(`="`)); err != nil {
				return err
			}
			if _, err := w.Write([]byte(attr.Val)); err != nil {
				return err
			}
			if _, err := w.Write([]byte(`"`)); err != nil {
				return err
			}
		}

		if _, err := w.Write([]byte(">")); err != nil {
			return err
		}

		// Recursively render child nodes
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if err := renderNode(w, c); err != nil {
				return err
			}
		}

		if _, err := w.Write([]byte("</")); err != nil {
			return err
		}
		if _, err := w.Write([]byte(n.Data)); err != nil {
			return err
		}

		_, err := w.Write([]byte(">"))
		return err

	case html.CommentNode:
		if _, err := w.Write([]byte("<!--")); err != nil {
			return err
		}
		if _, err := w.Write([]byte(n.Data)); err != nil {
			return err
		}
		if _, err := w.Write([]byte("-->")); err != nil {
			return err
		}
		return nil
	}
	return nil
}

func ParseVueSfcToHtmlNode(r io.Reader) (scriptNodes []*html.Node, templateNode *html.Node, styleNodes []*html.Node, err error) {
	doc, err := parseWithCaseSensitive(r)
	if err != nil {
		return nil, nil, nil, err
	}
	scriptNodes = htmlquery.Find(doc, "//script")
	templateNode = htmlquery.FindOne(doc, "//template")
	styleNodes = htmlquery.Find(doc, "//style")

	if len(scriptNodes) == 0 {
		return nil, nil, nil, xfmt.Errorf("script node not found")
	}

	return scriptNodes, templateNode, styleNodes, nil

}

func parseWithCaseSensitive(r io.Reader) (*html.Node, error) {
	root := &html.Node{
		Type: html.ElementNode,
		Data: "root",
	}

	z := newCasePreservingTokenizer(r)
	stack := []*html.Node{root}
	for {
		tt := z.Next()
		switch tt {
		case html.ErrorToken:
			if z.Err() == io.EOF {
				return root, nil
			}
			return nil, z.Err()

		case html.StartTagToken, html.SelfClosingTagToken:
			tn, hasAttr := z.TagName()
			node := &html.Node{
				Type: html.ElementNode,
				Data: string(tn),
			}

			if hasAttr {
				for {
					key, val, moreAttr := z.TagAttr()
					node.Attr = append(node.Attr, html.Attribute{
						Key: string(key),
						Val: string(val),
					})
					if !moreAttr {
						break
					}
				}
			}

			parent := stack[len(stack)-1]
			parent.AppendChild(node)

			if tt != html.SelfClosingTagToken {
				stack = append(stack, node)
			}
			continue

		case html.EndTagToken:
			stack = stack[:len(stack)-1]
			continue

		case html.TextToken:
			text := &html.Node{
				Type: html.TextNode,
				Data: string(z.Text()),
			}
			parent := stack[len(stack)-1]
			parent.AppendChild(text)
			continue

		case html.CommentToken:
			comment := &html.Node{
				Type: html.CommentNode,
				Data: string(z.Text()),
			}
			parent := stack[len(stack)-1]
			parent.AppendChild(comment)
			continue
		}
	}

}
