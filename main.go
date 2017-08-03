package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"syscall"

	docopt "github.com/docopt/docopt-go"
	"github.com/rabbit-ci/logstreamer"
)

var version = "0.0.1"

func main() {
	self := path.Base(os.Args[0])
	logger := log.New(os.Stderr, fmt.Sprintf("%s: ", self), 0)

	orPanic := func(err error) {
		if err == nil {
			return
		}
		logger.Panic(err)
	}

	usage := `Usage: prefixout [-dt] [-p PREFIX | --prefix PREFIX] -- COMMAND [ARGS ...]

	Prefixes stdout/stderr of a command. Nuff said.

Arguments:
	PREFIX		Prefix (defaults to "COMMAND: ")
	COMMAND		Command to exec
	ARGS		Command arguments
	-d			Differentiate stdout/stderr
	-t			Timestamp output`

	arguments, err := docopt.Parse(usage, nil, true, fmt.Sprintf("prefixout %s", version), true)
	orPanic(err)

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

	prefix := fmt.Sprintf("%s: ", exec_cmd)
	if arguments["PREFIX"] != nil {
		prefix = arguments["PREFIX"].(string)
	}

	out_prefix := ""
	err_prefix := ""
	if arguments["-d"].(bool) {
		err_prefix = "[stderr] "
	}

	prefix_flags := 0
	if arguments["-t"].(bool) {
		prefix_flags |= log.Ldate | log.Ltime
	}

	logout := log.New(os.Stdout, prefix, prefix_flags)
	logStreamerOut := logstreamer.NewLogstreamer(logout, out_prefix, false)
	defer logStreamerOut.Close()

	logerr := log.New(os.Stderr, prefix, prefix_flags)
	logStreamerErr := logstreamer.NewLogstreamer(logerr, err_prefix, true)
	defer logStreamerErr.Close()

	cmd := exec.Command(exec_cmd, exec_args...)
	cmd.Stdout = logStreamerOut
	cmd.Stderr = logStreamerErr

	// Reset any error
	logStreamerErr.FlushRecord()

	err = cmd.Start()
	if err != nil {
		log.Fatalf("ERROR! Could not spawn command. %v\n", err.Error())
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
			logger.Fatalf("cmd.Wait: %v", err)
		}
	}
}
