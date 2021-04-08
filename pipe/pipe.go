package pipe

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/magefile/mage/mg"
)

type PipedCmd struct {
	cmd      string
	args     []string
	env      []string
	readFrom *PipedCmd
	pipeTo   *PipedCmd
}

func NewPiped(cmd string, args ...string) *PipedCmd {
	return &PipedCmd{
		cmd:  cmd,
		args: args,
	}
}

func (p *PipedCmd) WithEnv(e []string) *PipedCmd {
	p.env = e
	return p
}

func (p *PipedCmd) Pipe(cmd string, args ...string) *PipedCmd {
	if p.readFrom != nil {
		panic("pipe already set to read")
	}
	if p.pipeTo != nil {
		panic("pipe already set to pipe to")
	}
	ret := &PipedCmd{
		cmd:      cmd,
		args:     args,
		readFrom: p,
	}
	p.pipeTo = ret
	return ret
}

func (p *PipedCmd) Run(ctx context.Context) error {
	return p.Execute(ctx, nil, os.Stdout, os.Stderr)
}

func (p *PipedCmd) Execute(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	cmdCtx, withCancel := context.WithCancel(ctx)
	defer withCancel()
	// Setup and start each command
	commands := make([]*exec.Cmd, 0)
	for current := p; current != nil; current = current.readFrom {
		//nolint:gosec
		cmd := exec.CommandContext(cmdCtx, current.cmd, current.args...)
		cmd.Stderr = stderr
		cmd.Env = current.env
		// put the last Pipe() at the first of commands
		commands = append([]*exec.Cmd{cmd}, commands...)
	}
	for idx := range commands {
		if mg.Verbose() {
			log.Println("Running command", commands[idx].Path, strings.Join(commands[idx].Args, " "))
		}
		if idx == 0 {
			commands[idx].Stdin = stdin
		} else {
			p, err := commands[idx-1].StdoutPipe()
			if err != nil {
				return fmt.Errorf("unable to get stdout pipe: %w", err)
			}
			commands[idx].Stdin = p
		}
		if idx == len(commands)-1 {
			commands[idx].Stdout = stdout
		}
	}
	for idx, cmd := range commands {
		if err := cmd.Start(); err != nil {
			withCancel()
			// Wait for the previous commands to finish so we do not leak
			for i := 0; i < idx; i++ {
				_ = commands[i].Wait()
			}
			return fmt.Errorf("unable to start command: %w", err)
		}
	}
	var waitErr error
	for i := len(commands) - 1; i >= 0; i-- {
		// https://golang.org/pkg/os/exec/#Cmd.StdoutPipe
		// "It is thus incorrect to call Wait before all reads from the pipe have completed"
		// So we need to Wait for the last in the chain first
		cmd := commands[i]
		if err := cmd.Wait(); err != nil {
			// We will end up returning the *last* wait error, which will be the first command of the pipes that failed
			waitErr = err
			withCancel()
		}
	}
	return waitErr
}
