package ui

import (
	"math/rand"
	"strconv"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/miwagner/socanui/canbus"
	"github.com/rivo/tview"
)

type TXView struct {
	cftx     *tview.Grid
	cftxF1   *tview.Form
	cftxF2   *tview.Form
	cftxData []*tview.InputField
}

// create txview
func (socanui *Socanui) createTXView() *TXView {
	txview := &TXView{}
	txview.cftxData = make([]*tview.InputField, 8)
	for i := 0; i < 8; i++ {
		txview.cftxData[i] = tview.NewInputField().SetLabelWidth(1).
			SetFieldWidth(3).
			SetAcceptanceFunc(tview.InputFieldMaxLength(2)).
			SetDoneFunc(func(key tcell.Key) {
			})
	}

	txview.cftxF1 = tview.NewForm().
		AddInputField("ID", "", 9, func(textToCheck string, lastChar rune) bool {
			i, err := strconv.ParseInt(textToCheck, 16, 64)
			if err != nil {
				return err == nil
			}
			if idx, _ := txview.cftxF1.GetFormItem(1).(*tview.DropDown).GetCurrentOption(); idx == 0 {
				if i >= 0x800 {
					return false
				}
			}
			if idx, _ := txview.cftxF1.GetFormItem(1).(*tview.DropDown).GetCurrentOption(); idx == 1 {
				if i >= 0x20000000 {
					return false
				}
			}
			return true
		}, nil).
		AddDropDown("Format", []string{"SFF", "EFF"}, 0, func(option string, optionIndex int) {

		}).
		AddCheckbox("RTR", false, func(checked bool) {
			for i := 0; i < 8; i++ {
				txview.cftxData[i].SetDisabled(true)
				txview.cftxData[i].SetFieldTextColor(tview.Styles.PrimitiveBackgroundColor)
			}
			l, err := strconv.Atoi(txview.cftxF1.GetFormItem(3).(*tview.InputField).GetText())
			if err == nil && !checked {
				for i := l; i < 8; i++ {
					txview.cftxData[i].SetDisabled(true)
					txview.cftxData[i].SetFieldTextColor(tview.Styles.PrimitiveBackgroundColor)
				}
				for i := 0; i < l; i++ {
					txview.cftxData[i].SetDisabled(false)
					txview.cftxData[i].SetFieldTextColor(tview.Styles.PrimaryTextColor)
				}
			}
		}).
		AddInputField("Length", "", 2, func(textToCheck string, lastChar rune) bool {
			i, err := strconv.Atoi(textToCheck)
			if err != nil || i > 8 {
				return false
			}
			return true
		}, func(text string) {
			l, err := strconv.Atoi(text)
			if err == nil && !txview.cftxF1.GetFormItem(2).(*tview.Checkbox).IsChecked() {
				for i := l; i < 8; i++ {
					txview.cftxData[i].SetDisabled(true)
					txview.cftxData[i].SetFieldTextColor(tview.Styles.PrimitiveBackgroundColor)
				}
				for i := 0; i < l; i++ {
					txview.cftxData[i].SetDisabled(false)
					txview.cftxData[i].SetFieldTextColor(tview.Styles.PrimaryTextColor)
				}
			}
		})
	txview.cftxF1.SetBorder(false)
	txview.cftxF1.SetHorizontal(true)
	txview.cftxF1.SetBorderPadding(0, 0, 0, 0)
	txview.cftxF1.GetFormItem(1).(*tview.DropDown).SetSelectedFunc(func(text string, index int) {
		txview.cftxF1.GetFormItem(0).(*tview.InputField).SetText("")
	})

	txview.cftxF2 = tview.NewForm().
		SetItemPadding(5).
		AddInputField("Period", "", 8, func(textToCheck string, lastChar rune) bool {
			_, err := strconv.Atoi(textToCheck)
			return err == nil
		}, nil).
		AddButton("Rep", func() {
			socanui.stopSend = false
			period, err := strconv.Atoi(txview.cftxF2.GetFormItem(0).(*tview.InputField).GetText())
			if err != nil {
				return
			}
			go func() {
				ticker := time.NewTicker(time.Millisecond * time.Duration(period))
				for range ticker.C {
					frame, err := socanui.createFrameFromView()
					if err == nil {
						socanui.sendFrame(*frame)
					}
					if socanui.stopSend {
						return
					}
				}
			}()
		}).
		AddButton("Rand", func() {
			rs := rand.NewSource(time.Now().UnixNano())
			r := rand.New(rs)
			socanui.stopSend = false
			period, err := strconv.Atoi(txview.cftxF2.GetFormItem(0).(*tview.InputField).GetText())
			if err != nil {
				return
			}
			go func() {
				ticker := time.NewTicker(time.Millisecond * time.Duration(period))
				for range ticker.C {
					frame := canbus.Frame{}
					if idx, _ := txview.cftxF1.GetFormItem(1).(*tview.DropDown).GetCurrentOption(); idx == 0 {
						frame.Kind = canbus.SFF
						frame.ID = uint32(r.Intn(0x800))
					}
					if idx, _ := txview.cftxF1.GetFormItem(1).(*tview.DropDown).GetCurrentOption(); idx == 1 {
						frame.Kind = canbus.EFF
						frame.ID = uint32(r.Intn(0x20000000))
					}
					length := r.Intn(9)
					data := make([]byte, length)
					for i := 0; i < length; i++ {
						data[i] = byte(r.Intn(0x100))
					}
					frame.Data = data
					socanui.sendFrame(frame)
					if socanui.stopSend {
						return
					}
				}
			}()
		}).
		AddButton("Stop", func() {
			socanui.stopSend = true
		})
	txview.cftxF2.SetBorder(false)
	txview.cftxF2.SetHorizontal(true)
	txview.cftxF2.SetBorderPadding(0, 0, 0, 0)

	txv := tview.NewTextView().
		SetDynamicColors(true).
		SetTextColor(tcell.ColorGreen).
		SetText("[::b]TX Frame")

	socanui.txIndicate = tview.NewTextView().SetDynamicColors(true)
	socanui.txIndicate.SetTextAlign(tview.AlignCenter)
	socanui.txIndicate.SetBackgroundColor(tcell.ColorGreen)
	socanui.txIndicate.SetTextColor(tcell.ColorWhite)

	btnOne := tview.NewButton("One")
	btnOne.SetSelectedFunc(func() {
		frame, err := socanui.createFrameFromView()
		if err == nil {
			socanui.sendFrame(*frame)
		}
	})

	txview.cftx = tview.NewGrid().
		SetRows(-1, -1, -1, 1).
		SetColumns(40, 4, 4, 4, 4, 4, 4, 4, 4, -5).
		SetBorders(false).
		AddItem(txview.cftxF1, 2, 0, 1, 1, 0, 0, false).
		AddItem(txview.cftxF2, 3, 0, 1, 3, 0, 0, false).
		AddItem(txv, 0, 0, 1, 1, 0, 0, false).
		AddItem(socanui.txIndicate, 3, 4, 1, 1, 0, 0, false).
		AddItem(btnOne, 3, 6, 1, 3, 0, 0, false)
	for i := 0; i < 8; i++ {
		txview.cftx.AddItem(txview.cftxData[i], 2, 1+i, 1, 1, 0, 0, false)
	}
	for i := 0; i < 8; i++ {
		a := tview.NewTextView().SetText(strconv.Itoa(i))
		a.SetTextAlign(tview.AlignCenter)
		txview.cftx.AddItem(a, 1, 1+i, 1, 1, 0, 0, false)
	}
	txview.cftx.SetBorderPadding(0, 0, 0, 0)

	return txview
}
