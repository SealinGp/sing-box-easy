package sublink

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"strings"

	"github.com/SealinGp/sing-box-easy/app/pkg/sublink/node"
	"github.com/SealinGp/sing-box-easy/app/pkg/sublink/protocol"
	"github.com/imroc/req/v3"
)

type SubLink struct {
}

func (l *SubLink) ListNodes(lines []string) ([]*node.SubNode, error) {
	nodes := make([]*node.SubNode, 0)
	for _, line := range lines {
		// 订阅链接
		if strings.Contains(line, "http") {
			modeNodes, err := l.fetchNodes(line)
			if err != nil {
				continue
			}
			nodes = append(nodes, modeNodes...)
			continue
		}

		//单个节点
		sub_node, err := l.parseNode(line)
		if err != nil {
			continue
		}
		nodes = append(nodes, sub_node)
	}

	return nodes, nil
}

func (l *SubLink) fetchNodes(sub_url string) ([]*node.SubNode, error) {
	resp, err := req.Get(sub_url)
	if err != nil {
		return nil, err
	}

	var sub_nodes []*node.SubNode
	respStr, err := resp.ToString()
	if err != nil {
		return nil, err
	}

	nodes, err := base64.StdEncoding.DecodeString(respStr)
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(nodes))
	for scanner.Scan() {
		line := scanner.Text()

		sub_node, err := l.parseNode(line)
		if err != nil {
			continue
		}

		sub_nodes = append(sub_nodes, sub_node)
	}
	return sub_nodes, nil
}

func (l *SubLink) parseNode(line string) (*node.SubNode, error) {
	parser, err := protocol.NewParser(line)
	if err != nil {
		return nil, err
	}

	sub_node, err := parser.Parse()
	if err != nil {
		return nil, err
	}

	return sub_node, nil
}
