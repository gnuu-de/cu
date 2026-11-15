//go:build linux

package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

var baudRates = map[int]uint32{
	50: unix.B50, 75: unix.B75, 110: unix.B110, 134: unix.B134, 150: unix.B150,
	200: unix.B200, 300: unix.B300, 600: unix.B600, 1200: unix.B1200, 1800: unix.B1800,
	2400: unix.B2400, 4800: unix.B4800, 9600: unix.B9600, 19200: unix.B19200,
	38400: unix.B38400, 57600: unix.B57600, 115200: unix.B115200, 230400: unix.B230400,
	460800: unix.B460800, 500000: unix.B500000, 576000: unix.B576000, 921600: unix.B921600,
}

func openSerial(o options) (*os.File, func() error, error) {
	rate, ok := baudRates[o.speed]
	if !ok {
		return nil, nil, fmt.Errorf("unsupported baud rate: %d", o.speed)
	}
	fd, err := unix.Open(o.line, unix.O_RDWR|unix.O_NOCTTY|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("open %s: %w", o.line, err)
	}
	closeFD := func(e error) (*os.File, func() error, error) { unix.Close(fd); return nil, nil, e }
	if err = unix.SetNonblock(fd, false); err != nil {
		return closeFD(err)
	}
	old, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return closeFD(err)
	}
	settings := *old
	settings.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON | unix.IXOFF | unix.IXANY
	settings.Oflag &^= unix.OPOST
	settings.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	settings.Cflag &^= unix.CSIZE | unix.PARENB | unix.PARODD | unix.CBAUD | unix.CRTSCTS
	settings.Cflag &^= unix.CSTOPB
	settings.Cflag |= unix.CS8 | unix.CREAD | unix.CLOCAL | rate
	settings.Ispeed = rate
	settings.Ospeed = rate
	if o.xonxoff {
		settings.Iflag |= unix.IXON | unix.IXOFF
	}
	if o.rtscts {
		settings.Cflag |= unix.CRTSCTS
	}
	switch o.parity {
	case "even":
		settings.Cflag &^= unix.CS8
		settings.Cflag |= unix.CS7 | unix.PARENB
		settings.Iflag |= unix.INPCK
	case "odd":
		settings.Cflag &^= unix.CS8
		settings.Cflag |= unix.CS7 | unix.PARENB | unix.PARODD
		settings.Iflag |= unix.INPCK
	}
	settings.Cc[unix.VMIN] = 1
	settings.Cc[unix.VTIME] = 0
	if err = unix.IoctlSetTermios(fd, unix.TCSETS, &settings); err != nil {
		return closeFD(err)
	}
	f := os.NewFile(uintptr(fd), o.line)
	restore := func() error { return unix.IoctlSetTermios(fd, unix.TCSETS, old) }
	return f, restore, nil
}

func sendBreak(f *os.File) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, f.Fd(), uintptr(unix.TCSBRK), 0)
	if errno != 0 {
		return errno
	}
	return nil
}
