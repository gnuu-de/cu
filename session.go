package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"golang.org/x/term"
)

// streamInput handles cu's line-initial escape sequences without consuming
// ordinary input beyond the bytes that have actually been sent to the line.
func streamInput(input io.Reader, remote io.Writer, local io.Writer, o options, breakSignal func() error) error {
	r := bufio.NewReader(input)
	atStart := true
	for {
		b, err := r.ReadByte()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if o.escapeEnabled && atStart && b == o.escape {
			cmd, err := r.ReadByte()
			if err != nil {
				return err
			}
			switch cmd {
			case '.':
				return nil
			case o.escape:
				b = o.escape
			case '?':
				fmt.Fprint(local, "\r\n~. disconnect  ~~ literal escape  ~> FILE send file  ~# break  ~? help\r\n")
				continue
			case '#':
				if breakSignal != nil {
					if err := breakSignal(); err != nil {
						return err
					}
				}
				continue
			case '>':
				name := strings.Builder{}
				for {
					c, e := r.ReadByte()
					if e != nil && e != io.EOF {
						return e
					}
					if e == io.EOF || c == '\r' || c == '\n' {
						break
					}
					name.WriteByte(c)
				}
				filename := strings.TrimSpace(name.String())
				if filename == "" {
					return fmt.Errorf("~> requires a filename")
				}
				file, err := os.Open(filename)
				if err != nil {
					return err
				}
				_, copyErr := io.Copy(remote, file)
				closeErr := file.Close()
				if copyErr != nil {
					return copyErr
				}
				if closeErr != nil {
					return closeErr
				}
				atStart = true
				continue
			default:
				// Unknown escape commands are passed through to the remote end.
				if _, err = remote.Write([]byte{o.escape}); err != nil {
					return err
				}
				b = cmd
			}
		}
		if _, err = remote.Write([]byte{b}); err != nil {
			return err
		}
		if o.halfDuplex {
			if _, err = local.Write([]byte{b}); err != nil {
				return err
			}
		}
		atStart = b == '\r' || b == '\n'
	}
}

func copyOutput(out io.Writer, in io.Reader, mapCR bool) error {
	if !mapCR {
		_, err := io.Copy(out, in)
		return err
	}
	buf := make([]byte, 4096)
	lastCR := false
	for {
		n, err := in.Read(buf)
		if n > 0 {
			mapped := make([]byte, 0, n*2)
			for _, b := range buf[:n] {
				if lastCR && b == '\n' {
					lastCR = false
					continue
				}
				lastCR = b == '\r'
				mapped = append(mapped, b)
				if b == '\r' {
					mapped = append(mapped, '\n')
				}
			}
			if _, e := out.Write(mapped); e != nil {
				return e
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
	}
}

func runSession(o options, input *os.File, output io.Writer) error {
	serial, restore, err := openSerial(o)
	if err != nil {
		return err
	}
	defer func() { restore(); serial.Close() }()
	if term.IsTerminal(int(input.Fd())) {
		state, err := term.MakeRaw(int(input.Fd()))
		if err != nil {
			return fmt.Errorf("terminal: %w", err)
		}
		defer term.Restore(int(input.Fd()), state)
	}
	fmt.Fprint(os.Stderr, "Connected. Escape with ~. at the beginning of a line.\r\n")
	var writes sync.Mutex
	lockedOutput := lockedWriter{out: output, mutex: &writes}
	readDone := make(chan error, 1)
	go func() { readDone <- copyOutput(lockedOutput, serial, o.mapCR) }()
	writeDone := make(chan error, 1)
	go func() {
		writeDone <- streamInput(input, serial, lockedOutput, o, func() error { return sendBreak(serial) })
	}()
	select {
	case err = <-readDone:
		return err
	case err = <-writeDone:
		return err
	}
}

type lockedWriter struct {
	out   io.Writer
	mutex *sync.Mutex
}

func (w lockedWriter) Write(b []byte) (int, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.out.Write(b)
}
