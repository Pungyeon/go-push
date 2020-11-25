package model

import (
	"bufio"
	"fmt"
	"github.com/tmc/scp"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"strings"
)

type Command interface {
	Run(Host) error
}

type SCP struct {
	Filename string
	Destination string
}
var _ Command = SCP{}

type Bash struct {
	Commands []string `yaml:"commands"`
}

func (b Bash) Run(host Host) error {
	for _, cmd := range b.Commands {
		sess, err := host.Client.NewSession()
		if err != nil {
			panic(err)
		}
		defer sess.Close()

		modes := ssh.TerminalModes{
			ssh.ECHO: 0,
			ssh.TTY_OP_ISPEED: 14400,
			ssh.TTY_OP_OSPEED: 14400,
		}
		if err := sess.RequestPty("xterm", 80, 40, modes); err != nil {
			panic(err)
		}

		stdout, err := sess.StdoutPipe()
		if err != nil {
			panic(err)
		}

		stderr, err := sess.StderrPipe()
		if err != nil {
			panic(err)
		}

		stdin, err := sess.StdinPipe()
		if err != nil {
			panic(err)
		}
		go listenForPasswordPrompt(stdin, stdout, host.Password)
		go io.Copy(os.Stderr, stderr)

		fmt.Println("#> "+cmd)
		if err :=  sess.Run(cmd); err != nil {
			return err
		}
	}
	return nil
}

func listenForPasswordPrompt(stdin io.WriteCloser, stdout io.Reader, password string) {
	r := bufio.NewReader(stdout)
	var output []byte
	for {
		b, err := r.ReadByte()
		if err != nil {
			break
		}
		output = append(output, b)
		if b == byte('[') {
			str, err := r.ReadString(']')
			if err != nil {
				break
			}
			output = append(output, []byte(str)...)
			if strings.HasPrefix(str, "sudo") {
				rest, err := r.ReadString(':')
				if err != nil {
					break
				}
				output = append(output, []byte(rest)...)
				_, err = stdin.Write([]byte(password + "\n"))
				if err != nil {
					break
				}
			}
		}
	}
	fmt.Println(string(output))
}

var _ Command = Bash{}

func (s SCP) Run(host Host) error {
	f, err := os.Open(s.Filename)
	if err != nil {
		return err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return err
	}
	sess, err := host.Client.NewSession()
	if err != nil {
		return err
	}
	defer sess.Close()
	fmt.Printf("sending file: %s to %s\n", f.Name(), host.Client.RemoteAddr())
	return scp.Copy(stat.Size(), os.ModePerm, f.Name(), f, s.Destination, sess)
}
