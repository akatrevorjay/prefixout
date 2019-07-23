package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"syscall"

	docopt "github.com/docopt/docopt-go"
	"github.com/logrusorgru/aurora"
)

var version = "0.1.1"

var self = path.Base(os.Args[0])
var logger = log.New(os.Stderr, fmt.Sprintf("%s: ", self), 0)

func orPanic(err error) {
	if err == nil {
		return
	}
	logger.Panic(err)
}

func main() {
	var err error

	usage := `Usage: prefixout [-dtc] [-p PREFIX | --prefix PREFIX] -- COMMAND [ARGS ...]

	Prefixes stdout/stderr of a command. Nuff said.

Arguments:
	PREFIX		Prefix (defaults to "COMMAND: ")
	COMMAND		Command to exec
	ARGS		Command arguments
	-d			Differentiate stdout/stderr
	-t			Timestamp output
	-c			Color output`

	arguments, err := docopt.Parse(usage, nil, true, fmt.Sprintf("prefixout %s", version), true)
	orPanic(err)

	au := aurora.NewAurora(arguments["-c"].(bool))

	exec_args := make([]string, 0)
	if arguments["ARGS"] != nil {
		exec_args = append(exec_args, arguments["ARGS"].([]string)...)
	}

	if arguments["COMMAND"] == nil {
		err := errors.New("No command specified")
		orPanic(err)
	}
	exec_cmd := arguments["COMMAND"].(string)

	exec_all := make([]string, 0)
	exec_all = append(exec_all, exec_cmd)
	exec_all = append(exec_all, exec_args...)

	var prefix string
	if arguments["PREFIX"] != nil {
		prefix = arguments["PREFIX"].(string)
	} else {
		prefix = fmt.Sprintf("%s: ", exec_cmd)
	}

	prefix_flags := 0
	if arguments["-t"].(bool) {
		prefix_flags |= log.Ldate | log.Ltime
	}

	out_prefix := prefix
	err_prefix := prefix
	if arguments["-d"].(bool) {
		err_prefix += "[stderr] "
	}

	logout := log.New(os.Stdout, "", prefix_flags)
	slurperOut := NewSlurper(logout, out_prefix, func(prefix string, str string) string {
		return au.Bold(prefix).String() + str
	})
	defer slurperOut.Close()

	logerr := log.New(os.Stderr, "", prefix_flags)
	slurperErr := NewSlurper(logerr, err_prefix, func(prefix string, str string) string {
		return au.Bold(prefix).Red().String() + str
	})
	defer slurperErr.Close()

	cmd := exec.Command(exec_cmd, exec_args...)
	// Attach fds (stdin is handled by wait)
	cmd.Stdout = slurperOut
	cmd.Stderr = slurperErr

	err = cmd.Start()
	if err != nil {
		logger.Fatalf("ERROR! Could not spawn command. %v\n", err.Error())
	}

	err = cmd.Wait()

	if err != nil {
		if exiterr, ok := err.(*exec.ExitError); ok {
			// The program has exited with an exit code != 0

			// This works on both Unix and Windows. Although package
			// syscall is generally platform dependent, WaitStatus is
			// defined for both Unix and Windows and in both cases has
			// an ExitStatus() method with the same signature.
			if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
				code := status.ExitStatus()
				os.Exit(code)
			}
		} else {
			orPanic(err)
		}
	}
}

type Slurper struct {
	Logger    *log.Logger
	buf       *bytes.Buffer
	prefix    string
	formatter func(string, string) string
}

func NewSlurper(logger *log.Logger, prefix string, formatter func(string, string) string) *Slurper {
	l := &Slurper{
		Logger:    logger,
		buf:       bytes.NewBuffer([]byte("")),
		prefix:    prefix,
		formatter: formatter,
	}

	return l
}

func termHasColors() bool {
	// TODO Just check if tty and/or interactive.

	term := os.Getenv("TERM")

	if strings.HasSuffix(term, "color") || strings.HasPrefix(term, "xterm") {
		return true
	}

	return false
}

func (l *Slurper) Write(p []byte) (n int, err error) {
	if n, err = l.buf.Write(p); err != nil {
		return
	}

	err = l.OutputLines()
	return
}

func (l *Slurper) Close() error {
	if err := l.Flush(); err != nil {
		return err
	}
	l.buf = bytes.NewBuffer([]byte(""))
	return nil
}

func (l *Slurper) Flush() error {
	var p []byte

	count, err := l.buf.Read(p)
	if err != nil {
		return err
	}

	if count > 0 {
		l.out(string(p))
	}

	return nil
}

func (l *Slurper) OutputLines() error {
	for {
		line, err := l.buf.ReadString('\n')

		if len(line) > 0 {
			if strings.HasSuffix(line, "\n") {
				l.out(line)
			} else {
				// put back into buffer, it's not a complete line yet
				//  Close() or Flush() have to be used to flush out
				//  the last remaining line if it does not end with a newline
				if _, err := l.buf.WriteString(line); err != nil {
					return err
				}
			}
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}
	}

	return nil
}

func (l *Slurper) out(str string) {
	if len(str) < 1 {
		return
	}

	if l.formatter != nil {
		str = l.formatter(l.prefix, str)
	}

	l.Logger.Print(str)
}
