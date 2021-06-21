package sys

import (
	"bufio"
	"bytes"
	"fmt"
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
	r           io.Reader
	donec       chan error
	streamDonce chan struct{}
	err         error
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
	buf := &bytes.Buffer{}
	r := io.TeeReader(co.r, buf)
	defer func() { co.r = buf }()

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		fn(line)
	}

	return co
}

func (co *Output) String() string {
	if co.r == nil {
		return ""
	}
	data, _ := ioutil.ReadAll(co.r)
	if r, ok := co.r.(*bytes.Reader); ok {
		r.Seek(0, io.SeekStart)
	} else {
		co.r = bytes.NewReader(data)
	}
	return string(data)
}

// RunCommand background
func RunCommand(c string, opts ...CommandOpt) (out *Output) {
	out = &Output{donec: make(chan error, 1), streamDonce: make(chan struct{}, 1)}
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
	pr, pw := io.Pipe()
	out.r = pr
	go func() {
		defer close(out.streamDonce)
		defer pw.Close()
		io.Copy(pw, outReader)

	}()
	go func() {
		defer close(out.donec)
		out.donec <- cmd.Wait()
	}()

	return
}

// RunCommand
func RunCommandV2(c string, opts ...CommandOpt) (string, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "bash"
	}
	binary, lookErr := exec.LookPath(shell)
	if lookErr != nil {
		return "", lookErr
	}
	o := &cmdopt{}
	for _, fn := range opts {
		fn(o)
	}

	cmd := exec.Command(binary, "-c", c)
	cmd.Dir = o.Dir
	cmd.Env = append(os.Environ(), o.Env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%v %s", err, string(out))
	}
	return string(out), nil
}
