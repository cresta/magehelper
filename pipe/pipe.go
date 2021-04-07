package pipe

import (
	"context"
	"fmt"
	"io"
	"os/exec"
)

type PipedCmd struct {
	cmd      string
	args     []string
	readFrom *PipedCmd
	pipeTo   *PipedCmd
}

func NewPiped(cmd string, args ...string) *PipedCmd {
	return &PipedCmd{
		cmd:  cmd,
		args: args,
	}
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

func (p *PipedCmd) Execute(ctx context.Context, stdin io.Reader, stdout io.Writer, stderr io.Writer) error {
	cmdCtx, withCancel := context.WithCancel(ctx)
	defer withCancel()
	// Setup and start each command
	commands := make([]*exec.Cmd, 0)
	for current := p; current != nil; current = current.readFrom {
		cmd := exec.CommandContext(cmdCtx, current.cmd, current.args...)
		cmd.Stderr = stderr
		// put the last Pipe() at the first of commands
		commands = append([]*exec.Cmd{cmd}, commands...)
	}
	for idx := range commands {
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
		if err := cmd.Wait(); err != nil && waitErr == nil {
			waitErr = err
			withCancel()
		}
	}
	if waitErr != nil {
		return fmt.Errorf("unable to wait for commands to finish: %w", waitErr)
	}
	return nil
}
