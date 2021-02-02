package sys

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"syscall"
)

// Exec command and replace current process
func Exec(cmdstr string) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "bash"
	}
	binary, lookErr := exec.LookPath(shell)
	if lookErr != nil {
		panic(lookErr)
	}
	args := []string{binary, "-i", "-c", cmdstr}
	env := os.Environ()
	execErr := syscall.Exec(binary, args, env)
	if execErr != nil {
		panic(execErr)
	}
}

type cmdopt struct {
	Dir string
	Env []string
}

type CommandOpt func(*cmdopt)

func WithEnv(k, v string) CommandOpt {
	return func(o *cmdopt) {
		o.Env = append(o.Env, k+"="+v)
	}
}

func WithWd(w string) CommandOpt {
	return func(o *cmdopt) {
		o.Dir = w
	}
}

type Output struct {
	r     io.Reader
	donec chan error
	err   error
}

func (co *Output) Stream() io.Reader {
	return co.r
}

func (co *Output) Drain() {
	if co.r == nil {
		return
	}
	io.Copy(ioutil.Discard, co.r)
}

func (co *Output) Error() error {
	if err := <-co.donec; err != nil {
		co.err = err
	}
	return co.err
}

func (co *Output) Wait() *Output {
	if err := <-co.donec; err != nil {
		co.err = err
	}
	return co
}

func (co *Output) ForEachLine(fn func(string)) *Output {
	if co.r == nil {
		return co
	}
	scanner := bufio.NewScanner(co.Stream())
	buf := new(bytes.Buffer)
	for scanner.Scan() {
		line := scanner.Text()
		buf.WriteString(line + "\n")
		fn(line)
	}
	co.r = buf
	return co
}

func (co *Output) String() string {
	if co.r == nil {
		return ""
	}
	out, err := ioutil.ReadAll(co.r)
	if err == nil {
		co.r = bytes.NewReader(out)
	}
	return string(out)
}

// RunCommand background
func RunCommand(c string, opts ...CommandOpt) (out *Output) {
	out = &Output{donec: make(chan error, 1)}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "bash"
	}
	binary, lookErr := exec.LookPath(shell)
	if lookErr != nil {
		close(out.donec)
		out.err = lookErr
		return
	}
	o := &cmdopt{}
	for _, fn := range opts {
		fn(o)
	}

	cmd := exec.Command(binary, "-c", c)
	cmd.Dir = o.Dir
	cmd.Env = append(os.Environ(), o.Env...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		close(out.donec)
		out.err = err
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		close(out.donec)
		out.err = err
		return
	}
	outReader := io.MultiReader(stdout, stderr)
	if err = cmd.Start(); err != nil {
		close(out.donec)
		out.err = err
		return
	}
	go func() {
		defer close(out.donec)
		out.donec <- cmd.Wait()
	}()

	out.r = outReader
	return
}
