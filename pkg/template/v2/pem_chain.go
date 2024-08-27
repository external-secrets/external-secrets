/*
MIT License

Copyright (c) Microsoft Corporation.

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE

Original Author: Anish Ramasekar https://github.com/aramase
In: https://github.com/Azure/secrets-store-csi-driver-provider-azure/pull/332
*/

package template

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

const (
	errNilCert           = "certificate is nil"
	errFoundDisjunctCert = "found multiple leaf or disjunct certificates"
	errNoLeafFound       = "no leaf certificate found"
	errChainCycle        = "constructing chain resulted in cycle"
)

type node struct {
	cert     *x509.Certificate
	parent   *node
	isParent bool
}

func fetchCertChains(data []byte) ([]byte, error) {
	var newCertChain []*x509.Certificate
	var pemData []byte
	nodes, err := pemToNodes(data)
	if err != nil {
		return nil, err
	}

	// at the end of this computation, the output will be a single linked list
	// the tail of the list will be the root node (which has no parents)
	// the head of the list will be the leaf node (whose parent will be intermediate certs)
	// (head) leaf -> intermediates -> root (tail)
	for i := range nodes {
		for j := range nodes {
			// ignore same node to prevent generating a cycle
			if i == j {
				continue
			}
			// if ith node AuthorityKeyId is same as jth node SubjectKeyId, jth node was used
			// to sign the ith certificate
			if bytes.Equal(nodes[i].cert.AuthorityKeyId, nodes[j].cert.SubjectKeyId) {
				nodes[j].isParent = true
				nodes[i].parent = nodes[j]
				break
			}
		}
	}

	var foundLeaf bool
	var leaf *node
	for i := range nodes {
		if !nodes[i].isParent {
			if foundLeaf {
				return nil, errors.New(errFoundDisjunctCert)
			}
			// this is the leaf node as it's not a parent for any other node
			leaf = nodes[i]
			foundLeaf = true
		}
	}

	if leaf == nil {
		return nil, errors.New(errNoLeafFound)
	}

	processedNodes := 0
	// iterate through the directed list and append the nodes to new cert chain
	for leaf != nil {
		processedNodes++
		// ensure we aren't stuck in a cyclic loop
		if processedNodes > len(nodes) {
			return pemData, errors.New(errChainCycle)
		}
		newCertChain = append(newCertChain, leaf.cert)
		leaf = leaf.parent
	}

	for _, cert := range newCertChain {
		b := &pem.Block{
			Type:  pemTypeCertificate,
			Bytes: cert.Raw,
		}
		pemData = append(pemData, pem.EncodeToMemory(b)...)
	}
	return pemData, nil
}

func pemToNodes(data []byte) ([]*node, error) {
	nodes := make([]*node, 0)
	for {
		// decode pem to der first
		block, rest := pem.Decode(data)
		data = rest

		if block == nil {
			break
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, err
		}
		// this should not be the case because ParseCertificate should return a non nil
		// certificate when there is no error.
		if cert == nil {
			return nil, errors.New(errNilCert)
		}
		nodes = append(nodes, &node{
			cert:     cert,
			parent:   nil,
			isParent: false,
		})
	}
	return nodes, nil
}
