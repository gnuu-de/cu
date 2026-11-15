//go:build linux

package main

import (
	"strconv"
	"testing"
	"unsafe"

	"golang.org/x/sys/unix"
)

func TestSerialPseudoTerminal(t *testing.T) {
	master, err := unix.Open("/dev/ptmx", unix.O_RDWR|unix.O_NOCTTY|unix.O_CLOEXEC, 0)
	if err != nil {
		t.Skip(err)
	}
	defer unix.Close(master)
	var unlock int32
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(master), uintptr(unix.TIOCSPTLCK), uintptr(unsafe.Pointer(&unlock)))
	if errno != 0 {
		t.Fatal(errno)
	}
	n, err := unix.IoctlGetInt(master, unix.TIOCGPTN)
	if err != nil {
		t.Fatal(err)
	}
	name := "/dev/pts/" + strconv.Itoa(n)
	o := options{line: name, speed: 115200, parity: "none", rtscts: false, xonxoff: false}
	serial, restore, err := openSerial(o)
	if err != nil {
		t.Fatal(err)
	}
	defer serial.Close()
	defer restore()
	config, err := unix.IoctlGetTermios(int(serial.Fd()), unix.TCGETS)
	if err != nil {
		t.Fatal(err)
	}
	if config.Cflag&unix.CBAUD != unix.B115200 || config.Cflag&unix.CS8 != unix.CS8 {
		t.Fatalf("wrong serial settings: %+v", config)
	}
	if config.Iflag&(unix.IXON|unix.IXOFF) != 0 || config.Cflag&unix.CRTSCTS != 0 {
		t.Fatal("flow control still enabled")
	}
	if _, err = serial.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, 4)
	if _, err = unix.Read(master, buf); err != nil {
		t.Fatal(err)
	}
	if string(buf) != "ping" {
		t.Fatalf("got %q", buf)
	}
	if _, err = unix.Write(master, []byte("pong")); err != nil {
		t.Fatal(err)
	}
	if _, err = serial.Read(buf); err != nil {
		t.Fatal(err)
	}
	if string(buf) != "pong" {
		t.Fatalf("got %q", buf)
	}
}
