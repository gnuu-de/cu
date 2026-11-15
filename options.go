package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const help = `Usage: cu [options] dir
 -l, --line DEVICE    Serial device (e.g. /dev/ttyUSB0)
 -s, --speed BAUD     Baud rate (default 9600)
 -E, --escape CHAR    Escape character (default ~; empty disables it)
 -e, -o               Even or odd parity (both select none)
 --parity=MODE        none, even, odd
 -h, --halfduplex     Echo typed characters locally
 -f, --nortscts       Disable RTS/CTS hardware flow control
 --nostop             Disable XON/XOFF software flow control
 -t, --mapcr          Map incoming CR to CR/LF
 -v, --version        Print version
 --help               Print help

At the start of a line: ~. disconnects, ~~ sends a literal ~,
~> FILE sends a file, ~# sends a break and ~? lists commands.
Only direct serial devices are supported; UUCP system and phone dialing
are not implemented.
`

const version = "cu-go 0.1.0"

type options struct {
	line                               string
	speed                              int
	escape                             byte
	escapeEnabled                      bool
	parity                             string
	halfDuplex, rtscts, xonxoff, mapCR bool
	showHelp, showVersion              bool
}

func parseOptions(args []string) (options, error) {
	o := options{speed: 9600, escape: '~', escapeEnabled: true, parity: "none", rtscts: true, xonxoff: true}
	var positional []string
	even, odd := false, false
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		value := func(name string) (string, error) {
			if k := strings.IndexByte(arg, '='); k >= 0 {
				return arg[k+1:], nil
			}
			if i+1 >= len(args) {
				return "", fmt.Errorf("%s requires a value", name)
			}
			i++
			return args[i], nil
		}
		switch {
		case arg == "--help":
			o.showHelp = true
		case arg == "-v" || arg == "--version":
			o.showVersion = true
		case arg == "-h" || arg == "--halfduplex":
			o.halfDuplex = true
		case arg == "-f" || arg == "--nortscts":
			o.rtscts = false
		case arg == "--nostop":
			o.xonxoff = false
		case arg == "-t" || arg == "--mapcr":
			o.mapCR = true
		case arg == "-e":
			even = true
			o.parity = "even"
		case arg == "-o":
			odd = true
			o.parity = "odd"
		case arg == "-l" || arg == "--line" || strings.HasPrefix(arg, "--line="):
			v, err := value("--line")
			if err != nil {
				return o, err
			}
			o.line = v
		case arg == "-s" || arg == "--speed" || arg == "--baud" || strings.HasPrefix(arg, "--speed=") || strings.HasPrefix(arg, "--baud="):
			v, err := value("--speed")
			if err != nil {
				return o, err
			}
			o.speed, err = strconv.Atoi(v)
			if err != nil || o.speed <= 0 {
				return o, fmt.Errorf("invalid speed: %q", v)
			}
		case arg == "-E" || arg == "--escape" || strings.HasPrefix(arg, "--escape="):
			v, err := value("--escape")
			if err != nil {
				return o, err
			}
			o.escapeEnabled = v != ""
			if len(v) > 1 {
				return o, errors.New("escape must be a single byte")
			}
			if len(v) > 0 {
				o.escape = v[0]
			}
		case strings.HasPrefix(arg, "--parity="):
			o.parity = strings.TrimPrefix(arg, "--parity=")
			if o.parity != "none" && o.parity != "even" && o.parity != "odd" {
				return o, fmt.Errorf("invalid parity: %q", o.parity)
			}
			even, odd = false, false
		case len(arg) > 1 && arg[0] == '-' && strings.Trim(arg[1:], "0123456789") == "":
			v, err := strconv.Atoi(arg[1:])
			if err != nil || v <= 0 {
				return o, fmt.Errorf("invalid speed: %q", arg)
			}
			o.speed = v
		case strings.HasPrefix(arg, "-l") && len(arg) > 2:
			o.line = arg[2:]
		case strings.HasPrefix(arg, "-s") && len(arg) > 2:
			v, err := strconv.Atoi(arg[2:])
			if err != nil || v <= 0 {
				return o, fmt.Errorf("invalid speed: %q", arg)
			}
			o.speed = v
		case strings.HasPrefix(arg, "-E") && len(arg) > 2:
			if len(arg) != 3 {
				return o, errors.New("escape must be a single byte")
			}
			o.escape = arg[2]
		case strings.HasPrefix(arg, "-"):
			return o, fmt.Errorf("unsupported option: %s", arg)
		default:
			positional = append(positional, arg)
		}
	}
	if even && odd {
		o.parity = "none"
	}
	if o.showHelp || o.showVersion {
		return o, nil
	}
	if len(positional) > 1 || len(positional) == 1 && positional[0] != "dir" {
		return o, errors.New("only direct serial connections (dir) are supported")
	}
	if o.line == "" {
		return o, errors.New("a serial device is required (-l DEVICE)")
	}
	if !strings.HasPrefix(o.line, "/dev/") {
		o.line = "/dev/" + o.line
	}
	return o, nil
}
