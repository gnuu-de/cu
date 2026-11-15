package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestOptions(t *testing.T) {
	for _, tc := range []struct {
		args  []string
		line  string
		speed int
		err   bool
	}{
		{[]string{"-l", "ttyUSB0", "-s", "115200", "dir"}, "/dev/ttyUSB0", 115200, false},
		{[]string{"--line=/dev/ttyS0", "-9600"}, "/dev/ttyS0", 9600, false},
		{[]string{"-l", "ttyS0", "5551234"}, "", 0, true},
		{[]string{"-z", "system"}, "", 0, true},
		{[]string{"-s", "oops", "-l", "ttyS0"}, "", 0, true},
		{[]string{"-l", "ttyS0", "-e", "-o", "--nostop", "-f", "-E", ""}, "/dev/ttyS0", 9600, false},
	} {
		o, err := parseOptions(tc.args)
		if (err != nil) != tc.err {
			t.Fatalf("%v: %v", tc.args, err)
		}
		if err == nil && (o.line != tc.line || o.speed != tc.speed) {
			t.Fatalf("%v: %+v", tc.args, o)
		}
	}
}

func TestParityAndEscape(t *testing.T) {
	o, err := parseOptions([]string{"-l", "ttyS0", "-e", "-o", "-E", ""})
	if err != nil {
		t.Fatal(err)
	}
	if o.parity != "none" || o.escapeEnabled {
		t.Fatalf("unexpected options: %+v", o)
	}
	var remote bytes.Buffer
	if err = streamInput(strings.NewReader("~.\r"), &remote, &bytes.Buffer{}, o, nil); err != nil {
		t.Fatal(err)
	}
	if remote.String() != "~.\r" {
		t.Fatalf("unexpected remote data %q", remote.String())
	}
}

func TestEscapes(t *testing.T) {
	o := options{escape: '~', escapeEnabled: true}
	var remote, local bytes.Buffer
	if err := streamInput(strings.NewReader("hello\r~~test\r~?~#~.ignored"), &remote, &local, o, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	if got := remote.String(); got != "hello\r~test\r" {
		t.Fatalf("remote %q", got)
	}
	if !strings.Contains(local.String(), "disconnect") {
		t.Fatal("missing escape help")
	}
}

func TestMapCR(t *testing.T) {
	var output bytes.Buffer
	if err := copyOutput(&output, strings.NewReader("hello\r\nworld\r!"), true); err != nil {
		t.Fatal(err)
	}
	if got := output.String(); got != "hello\r\nworld\r\n!" {
		t.Fatalf("got %q", got)
	}
}

func TestSendFileAndContinue(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "upload")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = file.WriteString("file contents"); err != nil {
		t.Fatal(err)
	}
	if err = file.Close(); err != nil {
		t.Fatal(err)
	}
	var remote bytes.Buffer
	o := options{escape: '~', escapeEnabled: true}
	in := strings.NewReader("~> " + file.Name() + "\nmore\r~.")
	if err = streamInput(in, &remote, &bytes.Buffer{}, o, nil); err != nil {
		t.Fatal(err)
	}
	if got := remote.String(); got != "file contentsmore\r" {
		t.Fatalf("received %q", got)
	}
}
