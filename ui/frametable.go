package ui

import (
	"fmt"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/miwagner/socanui/canbus"
	"github.com/rivo/tview"
)

type FrameTable struct {
	cft  *tview.Frame
	cftT *tview.Table
}

type TableData struct {
	tview.TableContentReadOnly
}

type trow struct {
	id     uint32
	dlc    uint8
	data   []byte
	kind   canbus.Kind
	period int64
	last   int64
	count  uint64
	cell   *tview.TableCell
}

var lookuptable = make(map[uint32]int) // lookup table
var trows = make([]trow, 0)
var tabledata = &TableData{}

// create frame table
func (socanui *Socanui) createFrameTable() *FrameTable {
	frametable := &FrameTable{}
	frametable.cftT = tview.NewTable().
		SetBorders(false).
		SetContent(tabledata).
		SetSelectable(false, false)
	frametable.cft = tview.NewFrame(frametable.cftT).
		SetBorders(0, 0, 0, 0, 1, 1).
		AddText("ID       DLC  DATA                       Period  Count  ASCII", true, tview.AlignLeft, tcell.ColorWhite)

	frametable.cftT.SetFocusFunc(func() {
		frametable.cft.SetBackgroundColor(tcell.ColorGrey)
	})
	frametable.cftT.SetBlurFunc(func() {
		frametable.cft.SetBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	})
	return frametable
}

func newTRow(id uint32, dlc uint8, data []byte, kind canbus.Kind) trow {
	row := trow{
		id:     id,
		count:  1,
		data:   data,
		dlc:    dlc,
		kind:   kind,
		period: 0,
		last:   time.Now().UnixMilli(),
		cell:   tview.NewTableCell(""),
	}
	row.cell.SetText(row.cellText())
	row.cell.SetTextColor(tcell.ColorOrange)
	return row
}

func (row *trow) cellText() string {
	var data string
	for _, rd := range row.data {
		data += fmt.Sprintf("%02X ", rd)
	}
	if row.kind == canbus.RTR_SFF || row.kind == canbus.RTR_EFF {
		data = "---RTR---"
	}
	var id string
	if row.kind == canbus.SFF || row.kind == canbus.RTR_SFF {
		id = fmt.Sprintf("%03X", row.id)
	}
	if row.kind == canbus.EFF || row.kind == canbus.RTR_EFF {
		id = fmt.Sprintf("%08X", row.id)
	}
	return fmt.Sprintf("%-8s [%1d]  %-25s %7d %6d  |%-8s|", id, row.dlc, data, row.period, row.count, toASCII(row.data))
}

func (tdata *TableData) GetCell(row, column int) *tview.TableCell {
	if row < 0 || column > 0 {
		return nil
	}
	return trows[row].cell
}

func (tdata *TableData) GetRowCount() int {
	return len(trows)
}

func (tdata *TableData) GetColumnCount() int {
	return 1
}

func (tdata *TableData) Clear() {
	lookuptable = make(map[uint32]int)
	trows = make([]trow, 0)
}

func lookupTableUpdate() {
	lookuptable = make(map[uint32]int, len(trows))
	for i, tr := range trows {
		lookuptable[tr.id] = i
	}
}

func (tdata *TableData) InsertOrUpdateRow(id uint32, dlc uint8, data []byte, kind canbus.Kind) {
	// error frame
	if kind == canbus.ERR {
		return
	}
	row, found := lookuptable[id]
	if found {
		// update
		now := time.Now()
		trows[row].count++
		trows[row].data = data
		trows[row].dlc = dlc
		trows[row].period = now.UnixMilli() - trows[row].last
		trows[row].last = now.UnixMilli()
		trows[row].cell.SetText(trows[row].cellText())
	} else {
		// new row => ta elements == 0: insert by 0; ta elements == 1; insert by 0 or 1;
		// ta elements >= 2: insert by 0, between, end
		row := newTRow(id, dlc, data, kind)
		switch len(trows) {
		case 0: // start
			trows = make([]trow, 1)
			trows[0] = row
		case 1: // before or after
			if id < trows[0].id {
				trows = append([]trow{row}, trows...)
			} else {
				trows = append(trows, row)
			}
		default:
			if id < trows[0].id {
				trows = append([]trow{row}, trows...)
			}
			if id > trows[len(trows)-1].id {
				trows = append(trows, row)
			} else {
				for i, tr := range trows {
					if id > tr.id && id < trows[i+1].id {
						trows = append(trows, trow{})
						copy(trows[i+2:], trows[i+1:])
						trows[i+1] = row
						break
					}
				}
			}
		}
		lookupTableUpdate()
	}
}
