package main

import (
	"fmt"
	"github.com/dontpanic92/wxGo/wx"
)

func _on_list_online_gen(e wx.Event) {

}

func _on_list_online_dclick(e wx.Event) {

}

func _on_list_offline_gen(e wx.Event) {

}

func _on_list_offline_dclick(e wx.Event) {

}

func _on_options_button_click(e wx.Event) {

}

func _on_button_reload_channels_click(e wx.Event) {

}

func _on_button_quit(e wx.Event) {

}

func main() {
	fmt.Println("Hello, world!")
	app := wx.NewApp()
	frame := initMainStatusWindow()
	frame.Show()
	app.MainLoop()
	return
}
