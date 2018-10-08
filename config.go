// +build windows

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"text/tabwriter"

	"github.com/StackExchange/wmi"
	"github.com/pkg/errors"
)

type config struct {
	PNPDeviceID string `json:"pnp_device_id"`
	BaudRate    int    `json:"baud_rate"`
	LogDir      string `json:"log_dir"`
}

var configData *config

func loadConfig(path string) {
	_, err := os.Stat(path)
	if err != nil {
		panic("config.json not found")
	}

	if configData == nil {
		configData = new(config)
		buff, err := ioutil.ReadFile(path)
		if err != nil {
			panic(errors.Wrap(err, "Problem reading config file"))
		}
		err = json.Unmarshal(buff, configData)
		if err != nil {
			panic(errors.Wrap(err, "Problem unmarshaling config structure"))
		}
	}
}

func configPath() string {
	exepath, err := exePath()
	if err != nil {
		panic("Couldn't find executable path: " + err.Error())
	}

	return filepath.FromSlash(filepath.Dir(exepath) + "/config.json")
}

// InteractiveConfig prompts the user for the necessary config parameters
// and writes them to config.json
func interactiveConfig() error {
	var dst []serialPort
	client := &wmi.Client{AllowMissingFields: true}
	err := client.Query("SELECT DeviceID, MaxBaudRate, PNPDeviceID, Description FROM Win32_SerialPort", &dst)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
	fmt.Fprintln(w, "#\tDevice ID\tMax Baud Rate\tDescription")
	for i, device := range dst {
		fmt.Fprintf(w, "%d)\t%s\t%d\t%s\n", i+1, device.DeviceID, device.MaxBaudRate, device.Description)
	}
	w.Flush()

	fmt.Printf("\nEnter selection or c to cancel: ")

	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	deviceSelection := scanner.Text()

	deviceIndex, err := strconv.ParseInt(deviceSelection, 10, 64)
	if err != nil {
		fmt.Println("Cancelled")
		return nil
	}
	if int(deviceIndex) > len(dst) || deviceIndex < 1 {
		return fmt.Errorf("Invalid selection: %s", deviceSelection)
	}

	pnp := dst[deviceIndex-1].PNPDeviceID

	fmt.Printf("Enter baud rate: ")
	scanner.Scan()
	baudRateSelection := scanner.Text()

	baudRate, err := strconv.ParseInt(baudRateSelection, 10, 32)
	if err != nil {
		return fmt.Errorf("Invalid selection: %s", baudRateSelection)
	}

	currUser, _ := user.Current()
	path := filepath.ToSlash(currUser.HomeDir) + "/Saved Games/Frontier Developments/Elite Dangerous"
	pathInfo, err := os.Stat(filepath.FromSlash(path))
	isPathValid := err == nil && pathInfo.IsDir()
	if !isPathValid {
		fmt.Printf("Enter the Elite Dangerous log path: ")
	} else {
		fmt.Printf("\nDefault logs folder: %s\n", path)
		fmt.Printf("Enter a different one, or press enter to use the above: ")
	}

	scanner.Scan()
	logDirSelection := scanner.Text()

	logDir := ""
	if len(logDirSelection) > 2 {
		logDirInfo, err := os.Stat(filepath.FromSlash(logDirSelection))
		if err != nil || !logDirInfo.IsDir() {
			return fmt.Errorf("Couldn't find the selected log path: " + err.Error())
		}
		logDir = logDirSelection
	} else if isPathValid {
		logDir = path
	} else {
		return fmt.Errorf("Must enter a valid path")
	}

	conf := config{pnp, int(baudRate), logDir}
	confBytes, err := json.Marshal(conf)
	if err != nil {
		return fmt.Errorf("Couldn't marshal config into JSON: %s", err.Error())
	}

	err = ioutil.WriteFile(configPath(), confBytes, 0777)
	if err != nil {
		return fmt.Errorf("Couldn't write config.json file: %s", err.Error())
	}

	fmt.Println("Wrote config to " + filepath.ToSlash(configPath()))

	return nil
}

func getPNPDeviceID() string {
	if configData == nil {
		loadConfig(configPath())
	}
	return configData.PNPDeviceID
}

func getBaudRate() int {
	if configData == nil {
		loadConfig(configPath())
	}
	return configData.BaudRate
}

func getLogDir() string {
	if configData == nil {
		loadConfig(configPath())
	}
	return configData.LogDir
}
