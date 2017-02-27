
package main

import "github.com/rakslice/wxGo/wx"

// bring SimpleHtmlListBox into the namespace for use from a CustomWidget in the .wxg

type SimpleHtmlListBox wx.SimpleHtmlListBox

func NewSimpleHtmlListBox(args... interface{}) SimpleHtmlListBox {
	return wx.NewSimpleHtmlListBox(args...)
}
