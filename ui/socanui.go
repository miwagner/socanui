package ui

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/miwagner/socanui/canbus"
	"github.com/miwagner/socanui/candevice"
	"github.com/rivo/tview"
)

const (
	VERSION = "0.3"
	TITLE   = "SocketCAN User Interface"
)

type Socanui struct {
	app        *tview.Application
	candev     *candevice.CanDevice
	pages      *tview.Pages
	headBar    *tview.Flex
	frametable *FrameTable
	framelist  *FrameList
	txview     *TXView
	params     *tview.TextView
	statistics *tview.TextView
	buttonBar  *tview.TextView
	txIndicate *tview.TextView
	layout     *tview.Grid
	filter     *tview.Frame
	stopSend   bool
	blink      bool
}

// create the TView application
func CreateSocanUI(app *tview.Application, candev *candevice.CanDevice) {
	socanui := &Socanui{}
	socanui.app = app
	socanui.candev = candev

	// theme
	tview.Styles = tview.Theme{
		PrimitiveBackgroundColor:    tcell.Color236,
		ContrastBackgroundColor:     tcell.ColorBlue,
		MoreContrastBackgroundColor: tcell.ColorGreen,
		BorderColor:                 tcell.ColorWhite,
		TitleColor:                  tcell.ColorWhite,
		GraphicsColor:               tcell.ColorWhite,
		PrimaryTextColor:            tcell.ColorWhite,
		SecondaryTextColor:          tcell.ColorYellow,
		TertiaryTextColor:           tcell.ColorGreen,
		InverseTextColor:            tcell.ColorBlue,
		ContrastSecondaryTextColor:  tcell.ColorNavy,
	}

	// tview
	socanui.createApp()

	// CAN parameter
	socanui.parameter()

	// CAN statistic
	go socanui.statistic()

	// show CAN Frame receive
	go socanui.showCANreceive()

	// indicate CAN TX
	go socanui.indicateTX()
}

// create application
func (socanui *Socanui) createApp() {
	socanui.headBar = socanui.createHeadBar()
	socanui.frametable = socanui.createFrameTable()
	socanui.framelist = socanui.createFrameList()
	socanui.txview = socanui.createTXView()

	socanui.params = tview.NewTextView().
		SetDynamicColors(true).
		SetTextColor(tcell.ColorGreen)

	socanui.statistics = tview.NewTextView().
		SetDynamicColors(true).
		SetTextColor(tcell.ColorGreen).
		SetChangedFunc(func() {
			socanui.app.Draw()
		})

	socanui.createButtonBar()
	socanui.createFilterWindows()
	socanui.layout = socanui.createMainLayout()
	socanui.pages = socanui.createPages()
	socanui.pages.ShowPage("main")
	socanui.app.SetRoot(socanui.pages, true)
}

// create TX CAN frame from input fields
func (socanui *Socanui) createFrameFromView() (*canbus.Frame, error) {
	var formatIsSFF bool
	if idx, _ := socanui.txview.cftxF1.GetFormItem(1).(*tview.DropDown).GetCurrentOption(); idx == 0 {
		formatIsSFF = true
	}
	if idx, _ := socanui.txview.cftxF1.GetFormItem(1).(*tview.DropDown).GetCurrentOption(); idx == 1 {
		formatIsSFF = false
	}
	frame := canbus.Frame{}
	// format
	if formatIsSFF {
		frame.Kind = canbus.SFF
	} else {
		frame.Kind = canbus.EFF
	}
	// id
	id, err := strconv.ParseUint(socanui.txview.cftxF1.GetFormItem(0).(*tview.InputField).GetText(), 16, 64)
	if err != nil {
		return nil, err
	}
	frame.ID = uint32(id)
	// length
	length, err := strconv.Atoi(socanui.txview.cftxF1.GetFormItem(3).(*tview.InputField).GetText())
	if err != nil {
		return nil, err
	}
	// rtr
	if socanui.txview.cftxF1.GetFormItem(2).(*tview.Checkbox).IsChecked() {
		if formatIsSFF {
			frame.Kind = canbus.RTR_SFF
		} else {
			frame.Kind = canbus.RTR_EFF
		}
	}
	// data
	data := make([]byte, length)
	if frame.Kind == canbus.SFF || frame.Kind == canbus.EFF {
		for i := 0; i < length; i++ {
			nb, err := strconv.ParseInt(socanui.txview.cftxData[i].GetText(), 16, 64)
			if err != nil {
				return nil, err
			}
			data[i] = byte(nb)
		}
	}
	frame.Data = data
	return &frame, nil
}

// create main Layout
func (socanui *Socanui) createMainLayout() (layout *tview.Grid) {
	return tview.NewGrid().
		SetRows(1, -1, 5, 1).
		SetColumns(-25, -10, -15).
		SetBorders(true).
		AddItem(socanui.headBar, 0, 0, 1, 3, 0, 0, false).
		AddItem(socanui.frametable.cft, 1, 0, 1, 1, 0, 0, false).
		AddItem(socanui.framelist.cfl, 1, 1, 1, 2, 0, 0, false).
		AddItem(socanui.txview.cftx, 2, 0, 1, 1, 0, 0, false).
		AddItem(socanui.params, 2, 1, 1, 1, 0, 0, false).
		AddItem(socanui.statistics, 2, 2, 1, 1, 0, 0, false).
		AddItem(socanui.buttonBar, 3, 0, 1, 3, 0, 0, false)
}

// create pages
func (socanui *Socanui) createPages() (layout *tview.Pages) {
	return tview.NewPages().
		AddPage("main", socanui.layout, true, true).
		AddPage("help", socanui.createHelpWindows(), true, false).
		AddPage("parameter", socanui.createParameterWindows(), true, false).
		AddPage("filter", socanui.filter, false, false).
		AddPage("version", socanui.createVersionWindows(), true, false)
}

// show received CAN frames in the views
func (socanui *Socanui) showCANreceive() {
	for {
		msg, err := socanui.candev.RecFrame()
		if err != nil {
			log.Fatalf("recv error: %v\n", err)
		}
		// error frame
		if msg.Kind == canbus.ERR {
			log.Printf("*** Error frame: %v", msg)
			continue
		}
		// filter
		if socanui.candev.CanFilter.RangeActiv {
			if msg.ID < socanui.candev.CanFilter.IdStart || msg.ID > socanui.candev.CanFilter.IdEnd {
				continue
			}
		}
		// add list
		out := socanui.framelist.add(&msg)
		if len(out) > 0 {
			fmt.Fprint(socanui.framelist.cflV, out)
		}
		// update table
		socanui.app.QueueUpdate(func() {
			tabledata.InsertOrUpdateRow(msg.ID, uint8(len(msg.Data)), msg.Data, msg.Kind)
		})
	}
}

// show statistic in the view
func (socanui *Socanui) statistic() {
	stat := socanui.candev.CanStatstic
	stat.Runs = 1
	for range time.Tick(time.Second) {
		stat.RxFrameLastSec = stat.RxFrameSum - stat.RxFrameLast
		stat.RxFrameLast = stat.RxFrameSum

		stat.TxFrameLastSec = stat.TxFrameSum - stat.TxFrameLast
		stat.TxFrameLast = stat.TxFrameSum

		if stat.RxFrameLastSec > stat.RxFrameMaxSec {
			stat.RxFrameMaxSec = stat.RxFrameLastSec
		}
		if stat.TxFrameLastSec > stat.TxFrameMaxSec {
			stat.TxFrameMaxSec = stat.TxFrameLastSec
		}

		stat.RxFrameAveSec = stat.RxFrameSum / stat.Runs
		stat.TxFrameAveSec = stat.TxFrameSum / stat.Runs
		stat.Runs++

		out := "[blue::b]Statistics                 RX           TX[white::-]\n"
		out += fmt.Sprintf("%s%12d %12d\n", "Number of Frames:", stat.RxFrameSum, stat.TxFrameSum)
		out += fmt.Sprintf("%s%12d %12d\n", "Last Sec Frames: ", stat.RxFrameLastSec, stat.TxFrameLastSec)
		out += fmt.Sprintf("%s%12d %12d\n", "Max Frames/s:    ", stat.RxFrameMaxSec, stat.TxFrameMaxSec)
		out += fmt.Sprintf("%s%12d %12d\n", "Ave Frames/s:    ", stat.RxFrameAveSec, stat.TxFrameAveSec)

		socanui.statistics.SetText(out)
	}
}

// clear can statistic
func (socanui *Socanui) clearStatistic() {
	socanui.candev.CanStatstic.Runs = 1
	socanui.candev.CanStatstic.RxFrameLast = 0
	socanui.candev.CanStatstic.TxFrameLast = 0
	socanui.candev.CanStatstic.RxFrameLastSec = 0
	socanui.candev.CanStatstic.TxFrameLastSec = 0
	socanui.candev.CanStatstic.RxFrameAveSec = 0
	socanui.candev.CanStatstic.TxFrameAveSec = 0
	socanui.candev.CanStatstic.RxFrameSum = 0
	socanui.candev.CanStatstic.TxFrameSum = 0
	socanui.candev.CanStatstic.RxFrameMaxSec = 0
	socanui.candev.CanStatstic.TxFrameMaxSec = 0
}

// show parameters in the view
func (socanui *Socanui) parameter() {
	out := fmt.Sprintf("[blue::b]%s Parameters[white::-]\n", socanui.candev.CanInf)
	out += fmt.Sprintf("Bitrate:\t\t%d\n", socanui.candev.CanParams.Bitrate)
	out += fmt.Sprintf("State:\t\t\t%s\n", socanui.candev.CanParams.State)
	out += fmt.Sprintf("Restart in ms:\t%d\n", socanui.candev.CanParams.RestartTime)
	out += fmt.Sprintf("Sample Point:\t%.3f", socanui.candev.CanParams.SamplePoint)
	socanui.params.SetText(out)
}

// byte array to ascii
func toASCII(data []byte) string {
	ascii := make([]byte, len(data))
	copy(ascii, data)
	for i := range ascii {
		if ascii[i] < 32 || ascii[i] > 126 {
			ascii[i] = '.'
		}
	}
	return string(ascii)
}

// send CAN frame
func (socanui *Socanui) sendFrame(frame canbus.Frame) {
	socanui.blink = true
	socanui.candev.SendFrame(frame)
}

// indicate TX
func (socanui *Socanui) indicateTX() {
	for range time.Tick(time.Millisecond * 500) {
		text := ""
		if socanui.blink {
			text = "[::bl]TX"
			socanui.blink = false
		}
		if socanui.txIndicate.GetText(true) != text {
			socanui.app.QueueUpdate(func() {
				socanui.txIndicate.SetText(text)
			})
		}
	}
}

// create head bar
func (socanui *Socanui) createHeadBar() *tview.Flex {
	headfilter := tview.NewTextView().
		SetDynamicColors(true).
		SetTextColor(tcell.ColorRed).
		SetTextAlign(tview.AlignLeft)
	headtitle := tview.NewTextView().
		SetDynamicColors(true).
		SetTextColor(tcell.ColorWhite).
		SetTextAlign(tview.AlignCenter).
		SetText("[::b]" + TITLE)
	headinf := tview.NewTextView().
		SetDynamicColors(true).
		SetTextColor(tcell.ColorLightSteelBlue).
		SetTextAlign(tview.AlignRight).
		SetText(fmt.Sprintf("[:green:]%s[-:-:b] %s", strings.Join(socanui.candev.CanParams.Mode, ", "), socanui.candev.CanInf))
	return tview.NewFlex().
		AddItem(headfilter, 0, 2, false).
		AddItem(headtitle, 0, 5, true).
		AddItem(headinf, 0, 2, false)
}

// create button bar
func (socanui *Socanui) createButtonBar() {
	socanui.buttonBar = tview.NewTextView().
		SetTextColor(tcell.ColorRosyBrown).
		SetText("Ctrl+C Quit | Ctrl+F Filter | Ctrl+R Reset | Ctrl+P Parameter | Ctrl+V Version | Ctrl+H Help")

	socanui.app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlH {
			socanui.pages.ShowPage("help")
		}
		if event.Key() == tcell.KeyCtrlP {
			socanui.pages.ShowPage("parameter")
		}
		if event.Key() == tcell.KeyCtrlF {
			_, _, screenWidth, screenHeight := socanui.pages.GetRect()
			x := (screenWidth - 36) / 2
			y := (screenHeight - 20) / 2
			socanui.filter.SetRect(x, y, 36, 20)
			socanui.pages.ShowPage("filter")
		}
		if event.Key() == tcell.KeyCtrlV {
			socanui.pages.ShowPage("version")
		}
		if event.Key() == tcell.KeyCtrlR {
			socanui.clearStatistic()
			socanui.framelist.reset()
			socanui.frametable.cftT.Clear()
			socanui.framelist.cflV.Clear()
		}
		return event
	})
}
