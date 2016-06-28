// Package attributes provides helper functions for golang.org/x/net/html's node
// attributes.
package attr // import "github.com/saracen/fpf/attr"

import (
	"golang.org/x/net/html"
)

type Attributes []html.Attribute

func (attrs Attributes) Attribute(name string) *html.Attribute {
	for i, attr := range attrs {
		if attr.Key == name {
			return &attrs[i]
		}
	}
	return nil
}

func (attrs Attributes) Get(name string) string {
	if attr := attrs.Attribute(name); attr != nil {
		return attr.Val
	}
	return ""
}

func (attrs Attributes) Has(name string) bool {
	return attrs.Attribute(name) != nil
}

func (attrs Attributes) Remove(name string) {
	for i := range attrs {
		if attrs[i].Key == name {
			attrs = append(attrs[:i], attrs[i+1:]...)
		}
	}
}