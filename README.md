# cu-go

A Linux serial terminal client in Go, inspired by Taylor UUCP's `cu`.
It supports **direct serial connections** (`cu -l /dev/ttyUSB0 -s 115200 dir`),
not UUCP system names or telephone/modem dialing. The pre-existing
`cu_1.07-27build1_amd64.deb` in this directory is an original Ubuntu binary
used to inspect the command-line interface, not part of this Go build.

## Use

```sh
go build -o cu .
./cu -l /dev/ttyUSB0 -s 115200 dir
```

You may omit `dir` when specifying `-l`. Without an absolute path, `-l ttyS0`
resolves to `/dev/ttyS0`. The default speed is 9600. Available options are
listed by `cu --help`; `~.` at the start of a line disconnects, `~~` sends
a literal tilde, `~> FILE` sends a file, and `~#` sends a break.

Unlike Taylor UUCP cu, this implementation does not read UUCP configuration,
look up ports or dial systems. Advanced escape commands such as `~p`, `~t`,
`~<` and shell commands are not supported. The package is named `cu-go` and
conflicts with the system `cu` package because both install `/usr/bin/cu`.

Run `go test ./...` to test option parsing, escape commands and serial
input/output using a Linux pseudoterminal.

## Debian packages on GitHub

The [GitHub Actions workflow](.github/workflows/deb.yml) runs tests and builds
`amd64` and `arm64` `.deb` packages on pushes, pull requests and manual runs.
Each push to `main` updates the `development` prerelease with both packages.
A tag such as `v0.1.0` creates a versioned GitHub Release with both packages.

Build locally with Go and `dpkg-deb`:

```sh
bash scripts/build-deb.sh 0.1.0 amd64
```
