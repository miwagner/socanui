package candevice

import (
	"errors"
	"log"
	"net"
	"os/exec"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/miwagner/socanui/canbus"
)

type CanDevice struct {
	CanParams     *canParameter
	CanInterfaces *canInterfaces
	CanStatstic   *canStatistic
	CanInf        string
	Sck           *canbus.Socket
	CanFilter     *canFilter
}

type canParameter struct {
	Mode        []string
	Bitrate     uint64
	SamplePoint float64
	State       string
	RestartTime uint64
	Tq          uint64
	PropSeg     uint8
	PhaseSeg1   uint8
	PhaseSeg2   uint8
	Sjw         uint8
}

type canInterfaces struct {
	can  []string
	vcan []string
}

type canStatistic struct {
	RxFrameSum     uint64
	TxFrameSum     uint64
	RxFrameLastSec uint64
	TxFrameLastSec uint64
	RxFrameMaxSec  uint64
	TxFrameMaxSec  uint64
	RxFrameAveSec  uint64
	TxFrameAveSec  uint64
	RxFrameLast    uint64
	TxFrameLast    uint64
	Runs           uint64
}
type canFilter struct {
	IdStart    uint32
	IdEnd      uint32
	RangeActiv bool
}

func NewDevice(caninf string) (*CanDevice, error) {
	var err error

	canDev := &CanDevice{}
	canDev.CanInf = caninf

	canDev.CanInterfaces, err = getCanInterfaces()
	if err != nil {
		log.Println(err)
	}
	log.Println("Interfaces:")
	log.Println("Interfaces CAN: ", canDev.CanInterfaces.can)
	log.Println("Interfaces VCAN: ", canDev.CanInterfaces.vcan)

	//  Check if the interface exists
	if !(slices.Contains(canDev.CanInterfaces.can, canDev.CanInf) || slices.Contains(canDev.CanInterfaces.vcan, canDev.CanInf)) {
		err = errors.New("Interface not exists")
		return &CanDevice{}, err
	}

	// Check if the interface is up
	inf, err := net.InterfaceByName(canDev.CanInf)
	if err != nil {
		err = errors.New("Interface error")
		return &CanDevice{}, err
	}
	if !strings.Contains(inf.Flags.String(), "up") {
		err = errors.New("Interface is not up")
		return &CanDevice{}, err
	}

	log.Println("CAN Parameter:")
	canDev.CanParams = canDev.getCanParameter()

	log.Println("Mode: ", canDev.CanParams.Mode)
	log.Println("Bitrate: ", canDev.CanParams.Bitrate)
	log.Println("Samplepoint: ", canDev.CanParams.SamplePoint)
	log.Println("State:", canDev.CanParams.State)
	log.Println("Restart in ms:", canDev.CanParams.RestartTime)
	log.Println("TQ:", canDev.CanParams.Tq)
	log.Println("PropSeg:", canDev.CanParams.PropSeg)
	log.Println("PhaseSeg1:", canDev.CanParams.PhaseSeg1)
	log.Println("PhaseSeg2:", canDev.CanParams.PhaseSeg2)
	log.Println("Sjw:", canDev.CanParams.Sjw)

	canDev.CanStatstic = &canStatistic{}

	canDev.CanFilter = &canFilter{}

	return canDev, nil
}

func (candevice *CanDevice) Connect() error {
	var err error
	candevice.Sck, err = canbus.New()
	if err != nil {
		log.Fatal(err)
		return err
	}

	err = candevice.Sck.Bind(candevice.CanInf)
	if err != nil {
		log.Fatalf("error binding to [%s]: %v\n", candevice.CanInf, err)
		return err
	}
	return nil
}

func (candevice *CanDevice) RecFrame() (canbus.Frame, error) {
	msg, err := candevice.Sck.Recv()
	if err != nil {
		log.Fatalf("recv error: %v\n", err)
		return msg, err
	}
	candevice.CanStatstic.RxFrameSum++

	return msg, nil
}

func (candevice *CanDevice) SendFrame(frame canbus.Frame) error {
	_, err := candevice.Sck.Send(frame)
	if err != nil {
		log.Fatalf("error sending data: %v\n", err)
	}
	candevice.CanStatstic.TxFrameSum++

	return nil
}

func (canDev *CanDevice) getCanParameter() *canParameter {
	var err error
	canparameter := &canParameter{}

	// Only can interface, not vcan
	if !slices.Contains(canDev.CanInterfaces.can, canDev.CanInf) {
		return canparameter
	}

	output, err := exec.Command("ip", "-details", "link", "show", canDev.CanInf).Output()
	if err != nil {
		return canparameter
	}

	lines := strings.Split(strings.TrimSuffix(string(output), "\n"), "\n")[2:]
	for i, line := range lines {
		line = strings.TrimSpace(line)
		log.Println(line)
		// state and restart-ms
		if i == 0 {
			re := regexp.MustCompile("can (?P<mode>.*)state (?P<state>.*) restart-ms (?P<restartms>[0-9]*)")
			matches := re.FindStringSubmatch(line)
			log.Println(matches[re.SubexpIndex("mode")])
			log.Println(matches[re.SubexpIndex("state")])
			log.Println(matches[re.SubexpIndex("restartms")])

			mode := strings.TrimSpace(matches[re.SubexpIndex("mode")])
			mode = strings.TrimLeft(mode, "<")
			mode = strings.TrimRight(mode, ">")
			canparameter.Mode = strings.Split(mode, ",")
			canparameter.State = matches[re.SubexpIndex("state")]
			canparameter.RestartTime, err = strconv.ParseUint(matches[re.SubexpIndex("restartms")], 10, 64)
			if err != nil {
				return &canParameter{}
			}
			if canparameter.State == "STOPPED" {
				return &canParameter{}
			}
		}
		// bitrate and samplepoint
		if i == 1 {
			re := regexp.MustCompile("bitrate (?P<bitrate>[0-9]*) sample-point (?P<samplepoint>([0-9]*[.])?[0-9]+)")
			matches := re.FindStringSubmatch(line)
			if len(matches) == 4 {
				log.Println(matches[re.SubexpIndex("bitrate")])
				log.Println(matches[re.SubexpIndex("samplepoint")])
				canparameter.Bitrate, err = strconv.ParseUint(matches[re.SubexpIndex("bitrate")], 10, 32)
				if err != nil {
					return &canParameter{}
				}
				canparameter.SamplePoint, err = strconv.ParseFloat(matches[re.SubexpIndex("samplepoint")], 64)
				if err != nil {
					return &canParameter{}
				}
			}
		}
		// tq, prop-seg, phase-seg1, phase-seg2, swj
		if i == 2 {
			re := regexp.MustCompile("tq (?P<tq>[0-9]*) prop-seg (?P<propseg>([0-9])) phase-seg1 (?P<phaseseg1>([0-9])) phase-seg2 (?P<phaseseg2>([0-9])) sjw (?P<sjw>([0-9]))")
			matches := re.FindStringSubmatch(line)
			if len(matches) == 10 {
				log.Println(matches[re.SubexpIndex("tq")])
				log.Println(matches[re.SubexpIndex("propseg")])
				log.Println(matches[re.SubexpIndex("phaseseg1")])
				log.Println(matches[re.SubexpIndex("phaseseg2")])
				log.Println(matches[re.SubexpIndex("sjw")])

				canparameter.Tq, err = strconv.ParseUint(matches[re.SubexpIndex("tq")], 10, 32)
				if err != nil {
					return &canParameter{}
				}
				t, err := strconv.ParseUint(matches[re.SubexpIndex("propseg")], 10, 8)
				if err != nil {
					return &canParameter{}
				}
				canparameter.PropSeg = uint8(t)
				t, _ = strconv.ParseUint(matches[re.SubexpIndex("phaseseg1")], 10, 8)
				if err != nil {
					return &canParameter{}
				}
				canparameter.PhaseSeg1 = uint8(t)
				t, _ = strconv.ParseUint(matches[re.SubexpIndex("phaseseg2")], 10, 8)
				if err != nil {
					return &canParameter{}
				}
				canparameter.PhaseSeg2 = uint8(t)
				t, _ = strconv.ParseUint(matches[re.SubexpIndex("sjw")], 10, 8)
				if err != nil {
					return &canParameter{}
				}
				canparameter.Sjw = uint8(t)
				break
			}
		}
	}
	return canparameter
}

func getCanInterfaces() (*canInterfaces, error) {
	var err error

	ci := &canInterfaces{}
	re := regexp.MustCompile(`\d+:\s(?P<inf>\w+):`)

	// Type can
	output, err := exec.Command("ip", "link", "show", "type", "can").Output()
	if err != nil {
		log.Println(err)
		return &canInterfaces{}, err
	}
	lines := strings.Split(strings.TrimSuffix(string(output), "\n"), "\n")[:]
	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if matches != nil {
			ci.can = append(ci.can, matches[re.SubexpIndex("inf")])
		}
	}

	// Type vcan
	output, err = exec.Command("ip", "link", "show", "type", "vcan").Output()
	if err != nil {
		log.Println(err)
		return &canInterfaces{}, err
	}
	lines = strings.Split(strings.TrimSuffix(string(output), "\n"), "\n")[:]
	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if matches != nil {
			ci.vcan = append(ci.vcan, matches[re.SubexpIndex("inf")])
		}
	}
	return ci, nil
}
