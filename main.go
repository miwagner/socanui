package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/miwagner/socanui/candevice"
	"github.com/miwagner/socanui/ui"
	"github.com/rivo/tview"
)

const (
	DEFAULT_INTERFACE = "vcan0"
)

func main() {

	uselog := flag.Bool("l", false, "log file")
	usehelp := flag.Bool("h", false, "help")
	useversion := flag.Bool("v", false, "version")
	flag.Parse()
	log.SetOutput(io.Discard)
	if *uselog {
		file, err := os.OpenFile("socanui.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			log.Fatal(err)
		}
		log.SetOutput(file)
	}
	if *usehelp {
		help()
		os.Exit(0)
	}
	if *useversion {
		println(ui.VERSION)
		os.Exit(0)
	}
	caninf := DEFAULT_INTERFACE
	args := flag.Args()
	if len(args) == 1 {
		caninf = args[0]
	}
	if len(args) > 1 {
		help()
		os.Exit(1)
	}
	log.Printf("Interface: %s", caninf)

	// CAN bus
	candev, err := candevice.NewDevice(caninf)
	if err != nil {
		log.Println(err)
		fmt.Printf("Error: %v\n", err)
		fmt.Println("########################################")
		fmt.Println("You can add a virtual CAN interface:")
		fmt.Println("sudo modprobe vcan")
		fmt.Println("sudo ip link add dev vcan0 type vcan")
		fmt.Println("sudo ip link set up vcan0")
		fmt.Print("\nYou can generate testdata as follow:\n")
		fmt.Println("cangen vcan0")
		fmt.Println("########################################")
		os.Exit(1)
	}

	// CAN connect
	err = candev.Connect()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer candev.Sck.Close()

	// tview application
	app := tview.NewApplication()
	defer app.Stop()

	// create ui
	ui.CreateSocanUI(app, candev)

	if err = app.EnableMouse(true).Run(); err != nil {
		panic(err)
	}
}

// help
func help() {
	fmt.Println(`
SocketCAN User Interface (socanui)

Usage:
socanui [options] interface

Interface:
SocketCAN Interface such as "can0", "vcan0", "slcan0"

Options:
  -l            log debug to file "socanui.log"
  -h            display this help and exit
  -v            output version information and exit
  
Examples:
socanui can0
     (connect to can0 interface)
socanui -l vcan0
     (connect to vcan0 interface and write debug log)
	`)
}
