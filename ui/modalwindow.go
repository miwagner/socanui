package ui

import (
	"fmt"
	"strconv"

	"github.com/rivo/tview"
)

// create parameter windows
func (socanui *Socanui) createParameterWindows() *tview.Modal {
	parametertext := "[black::ub]CAN Parameter\n\n"
	parametertext += fmt.Sprintf("[black]Bitrate:          [white] %12d\n", socanui.candev.CanParams.Bitrate)
	parametertext += fmt.Sprintf("[black]State:            [white] %12s\n", socanui.candev.CanParams.State)
	parametertext += "[black]Mode:"
	if len(socanui.candev.CanParams.Mode) > 0 {
		for i, mode := range socanui.candev.CanParams.Mode {
			if i == 0 {
				parametertext += fmt.Sprintf("             [white] %12s\n", mode)
			} else {
				parametertext += fmt.Sprintf("                  [white] %12s\n", mode)
			}
		}
	} else {
		parametertext += "\n"
	}
	parametertext += fmt.Sprintf("[black]Samplepoint:      [white] %12.3f\n", socanui.candev.CanParams.SamplePoint)
	parametertext += fmt.Sprintf("[black]Restart ms:       [white] %12d\n", socanui.candev.CanParams.RestartTime)
	parametertext += fmt.Sprintf("[black]TQ:               [white] %12d\n", socanui.candev.CanParams.Tq)
	parametertext += fmt.Sprintf("[black]Prop-Seg:         [white] %12d\n", socanui.candev.CanParams.PropSeg)
	parametertext += fmt.Sprintf("[black]Phase-Seg-1:      [white] %12d\n", socanui.candev.CanParams.PhaseSeg1)
	parametertext += fmt.Sprintf("[black]Phase-Seg-2:      [white] %12d\n", socanui.candev.CanParams.PhaseSeg2)
	parametertext += fmt.Sprintf("[black]SJW:              [white] %12d\n", socanui.candev.CanParams.Sjw)

	parameterWindow := tview.NewModal().
		SetText(parametertext).
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Close" {
				socanui.pages.SwitchToPage("main")
			}
		})
	return parameterWindow
}

// create version windows
func (socanui *Socanui) createVersionWindows() *tview.Modal {
	versionWindow := tview.NewModal().
		SetText("[black::ub]SocketCAN User Interface\n\nhttps://github.com/miwagner/socanui\n\nMichael Wagner\n\nVersion " + VERSION + "\n").
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Close" {
				socanui.pages.SwitchToPage("main")
			}
		})
	return versionWindow
}

// create help windows
func (socanui *Socanui) createHelpWindows() *tview.Modal {
	helpWindow := tview.NewModal().
		SetText("[black::ub]Help\n\n[black]Navigate:  [white]Arrow Keys, Page Up/Down, Home, End").
		AddButtons([]string{"Close"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if buttonLabel == "Close" {
				socanui.pages.SwitchToPage("main")
			}
		})
	return helpWindow
}

// create filter windows
func (socanui *Socanui) createFilterWindows() {
	filterForm := tview.NewForm()
	filterForm.AddInputField("Start ID", "", 9, func(textToCheck string, lastChar rune) bool {
		_, err := strconv.ParseInt(textToCheck, 16, 64)
		return err == nil
	}, nil)
	filterForm.AddInputField("End ID", "", 9, func(textToCheck string, lastChar rune) bool {
		_, err := strconv.ParseInt(textToCheck, 16, 64)
		return err == nil
	}, nil)
	filterForm.AddCheckbox("Enable Range Filter", false, nil)
	filterForm.AddButton("OK", func() {
		// filter
		id, err := strconv.ParseUint(filterForm.GetFormItem(0).(*tview.InputField).GetText(), 16, 64)
		if err == nil {
			socanui.candev.CanFilter.IdStart = uint32(id)
		}
		id, err = strconv.ParseUint(filterForm.GetFormItem(1).(*tview.InputField).GetText(), 16, 64)
		if err == nil {
			socanui.candev.CanFilter.IdEnd = uint32(id)
		}
		socanui.candev.CanFilter.RangeActiv = filterForm.GetFormItem(2).(*tview.Checkbox).IsChecked()
		if filterForm.GetFormItem(2).(*tview.Checkbox).IsChecked() {
			socanui.headBar.GetItem(0).(*tview.TextView).SetText("[::b]Filter active")
		} else {
			socanui.headBar.GetItem(0).(*tview.TextView).SetText("")
		}
		socanui.pages.SwitchToPage("main")
	})

	gf := tview.NewFlex().
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(tview.NewTextView().SetText("Range Filter"), 1, 1, false).
			AddItem(filterForm, 0, 5, true), 0, 1, true)

	socanui.filter = tview.NewFrame(gf)
	socanui.filter.SetBorder(true).SetTitle("Filter")
}
