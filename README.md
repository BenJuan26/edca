# Elite Dangerous Cockpit Agent

Sends ED status updates via serial port

## Description

The monitor will watch the log files written by Elite Dangerous and write the relevant information through Serial to custom controllers or other devices.

## Build

Like most Go projects, building is simple:

```
go get
go build
```

## Usage

First, set up the configuration file. Either copy/rename `config-sample.json` and fill in the values appropriately, or use the interactive config by running `edca.exe configure`. The interactive config will prompt for the config values and will look like this:

```
#  Device ID                Max Baud Rate  Description
1  USB\VID_4321&PID_0001\1  115200         Communications Port

Enter selection or c to cancel: 1
Enter baud rate: 115200

Default logs folder: C:/Users/MyUser/Saved Games/Frontier Developments/Elite Dangerous
Enter a different one, or press enter to use the above: 

Wrote config to config.json
```

The service will now be ready to be installed with `edca.exe install`. Once it's installed, it can be stopped/started either from the Windows Services app or through the command line with `edca.exe [start|stop]`.
