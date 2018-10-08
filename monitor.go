// +build windows

package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/BenJuan26/elite"
	"github.com/StackExchange/wmi"
	"github.com/tarm/serial"
)

type controllerInfo struct {
	Timestamp  string   `json:"timestamp"`
	Flags      uint32   `json:"Flags"`
	Pips       [3]int32 `json:"Pips"`
	FireGroup  int32    `json:"FireGroup"`
	StarSystem string   `json:"StarSystem"`
}

type serialPort struct {
	MaxBaudRate int
	DeviceID    string
	Description string
	PNPDeviceID string
}

type errorNoSerialConnection struct {
	message string
}

func (e *errorNoSerialConnection) Error() string {
	return e.message
}

func getSerialPort(pnp string) (*serial.Port, error) {
	var dst []serialPort

	// WMI needs the backslashes to be escaped
	escaped := strings.Replace(pnp, "\\", "\\\\", -1)
	query := "SELECT DeviceID, MaxBaudRate FROM Win32_SerialPort WHERE PNPDeviceID='" + escaped + "'"
	client := &wmi.Client{AllowMissingFields: true}
	err := client.Query(query, &dst)
	if err != nil {
		return nil, fmt.Errorf("Couldn't connect to serial port: %s", err.Error())
	} else if len(dst) < 1 {
		return nil, fmt.Errorf("Couldn't find a PNP Device with ID '%s'", pnp)
	}

	conf := &serial.Config{Name: dst[0].DeviceID, Baud: getBaudRate()}
	s, err := serial.OpenPort(conf)
	if err != nil {
		return nil, fmt.Errorf("Couldn't open serial port: %s", err.Error())
	}

	elog.Info(1, fmt.Sprintf("Connected to serial port %s at baud rate %d", dst[0].DeviceID, getBaudRate()))
	return s, nil
}

func isSerialConnected() bool {
	var dst []serialPort
	escaped := strings.Replace(getPNPDeviceID(), "\\", "\\\\", -1)
	query := "SELECT DeviceID FROM Win32_SerialPort WHERE PNPDeviceID='" + escaped + "'"
	client := &wmi.Client{AllowMissingFields: true}
	client.Query(query, &dst)

	if len(dst) < 1 {
		return false
	}

	return true
}

var errorCount = 0
var lastStatus = &elite.Status{}
var lastSystem = ""
var s *serial.Port

func startWaitingForSerialDevice() {
	s = nil
	lastStatus = &elite.Status{}
	lastSystem = ""
	elog.Info(1, "Couldn't write to serial port")
	elog.Info(1, "Going to sleep until there is a serial connection (checking every 10 seconds)")
}

var count = 0

func checkStatusAndUpdate() error {
	if errorCount > 20 {
		return fmt.Errorf("Too many consecutive errors")
	}

	// Check in on the serial device every once in a while
	if count > 50 {
		count = 0
		if !isSerialConnected() {
			startWaitingForSerialDevice()
			return &errorNoSerialConnection{"Device not connected"}
		}
	}

	var err error
	if s == nil {
		s, err = getSerialPort(getPNPDeviceID())
		if err != nil {
			return &errorNoSerialConnection{err.Error()}
		}

	}

	status, err := elite.GetStatusFromPath(filepath.FromSlash(getLogDir()))
	if err != nil {
		errorCount = errorCount + 1
		elog.Error(1, "Couldn't get status: "+err.Error())
		if errorCount > 1 {
			elog.Error(1, fmt.Sprintf("Now at %d consecutive errors", errorCount))
		}
		return nil
	}

	system, err := elite.GetStarSystemFromPath(filepath.FromSlash(getLogDir()))
	if err != nil {
		errorCount = errorCount + 1
		elog.Error(1, "Couldn't get star system: "+err.Error())
		if errorCount > 1 {
			elog.Error(1, fmt.Sprintf("Now at %d consecutive errors", errorCount))
		}
		return nil
	}

	if status.Timestamp != lastStatus.Timestamp || lastSystem != system {
		lastStatus = status
		lastSystem = system

		info := controllerInfo{
			Timestamp:  status.Timestamp,
			Flags:      status.Flags,
			Pips:       status.Pips,
			FireGroup:  status.FireGroup,
			StarSystem: system,
		}

		infoBytes, err := json.Marshal(info)
		if err != nil {
			errorCount = errorCount + 1
			elog.Error(1, "Couldn't marshal JSON to send to serial: "+err.Error())
			if errorCount > 1 {
				elog.Error(1, fmt.Sprintf("Now at %d consecutive errors", errorCount))
			}
			return nil
		}

		_, err = s.Write(infoBytes)
		if err != nil {
			errorCount = errorCount + 1
			startWaitingForSerialDevice()
			return nil
		}
	}

	errorCount = 0
	count = count + 1
	return nil
}
