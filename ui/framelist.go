package ui

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/miwagner/socanui/canbus"
	"github.com/rivo/tview"
)

type FrameList struct {
	cfl  *tview.Frame
	cflV *tview.TextView
	out  string
	last int64
	br   string
}

const (
	DIFFVIEWMS = 250
)

// create frame list
func (socanui *Socanui) createFrameList() *FrameList {
	framelist := &FrameList{}
	framelist.cflV = tview.NewTextView().
		SetTextColor(tcell.ColorYellow).
		SetScrollable(false).
		SetMaxLines(100).
		SetChangedFunc(func() {
			socanui.app.Draw()
		})

	framelist.cfl = tview.NewFrame(framelist.cflV).
		SetBorders(0, 0, 0, 0, 1, 1).
		AddText("ID       DLC  DATA                       ASCII", true, tview.AlignLeft, tcell.ColorWhite)

	return framelist
}

func (framelist *FrameList) add(msg *canbus.Frame) string {
	var data string
	now := time.Now().UnixMilli()
	for _, t := range msg.Data {
		data += fmt.Sprintf("%02X ", t)
	}
	var id string
	if msg.Kind == canbus.SFF || msg.Kind == canbus.RTR_SFF {
		id = fmt.Sprintf("%03X", msg.ID)
	}
	if msg.Kind == canbus.EFF || msg.Kind == canbus.RTR_EFF {
		id = fmt.Sprintf("%08X", msg.ID)
	}
	if msg.Kind == canbus.RTR_SFF || msg.Kind == canbus.RTR_EFF {
		data = "---RTR---"
	}
	framelist.out += fmt.Sprintf("%s%-8s [%d]  %-25s  |%-8s|", framelist.br, id, len(msg.Data), data, toASCII(msg.Data))
	framelist.br = "\n"
	if now-framelist.last >= DIFFVIEWMS {
		outret := framelist.out
		framelist.last = now
		framelist.out = ""
		return outret
	}
	return ""
}

func (framelist *FrameList) reset() {
	framelist.br = ""
	framelist.out = ""
	framelist.last = 0
}
