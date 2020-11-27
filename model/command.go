package model

import (
	"bufio"
	"fmt"
	"github.com/tmc/scp"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

type Command interface {
	Run(Host) error
}

type Upload struct {
	Template bool `yaml:"template"`
	Filename string `yaml:"filename"`
	Destination string `yaml:"destination"`
}
var _ Command = Upload{}

type Bash struct {
	Commands []string `yaml:"commands"`
}

func (b Bash) Run(host Host) error {
	return b.WithTemplate(host).run(host)
}

func (b Bash) WithTemplate(host Host) Bash {
	for i := range b.Commands {
		b.Commands[i] = Template(host.Variables, b.Commands[i])
	}
	return b
}

func (b Bash) run(host Host) error {
	for _, cmd := range b.Commands {
		sess, err := host.Client.NewSession()
		if err != nil {
			return err
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
		//go io.Copy(os.Stdout, stdout)

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
				//fmt.Println("writing password to stdin")
				time.Sleep(time.Millisecond*100)
				_, err = stdin.Write([]byte(password + "\n"))
				if err != nil {
					fmt.Println(err)
					break
				}
			}
		}
	}
	fmt.Println(string(output))
}

var _ Command = Bash{}

func (s Upload) WithTemplate(host Host) Upload {
	return Upload {
		Template: s.Template,
		Filename: Template(host.Variables, s.Filename),
		Destination: Template(host.Variables, s.Destination),
	}
}

func (s Upload) Run(host Host) error {
	return s.WithTemplate(host).run(host)
}

func (s Upload) run(host Host) error {
	var f *os.File
	var err error
	if !s.Template {
		f, err = os.Open(s.Filename)
		if err != nil {
			return err
		}
	} else {
		tmp, err := RewriteTemplateFile(host, s.Filename)
		if err != nil {
			return err
		}
		defer func() {
			if err := os.Remove(tmp); err != nil {
				fmt.Println(err)
			}
		}()
		f, err = os.Open(tmp)
		if err != nil {
			return err
		}
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
	fmt.Printf("sending file (template: %v): %s (%s) -> %s:%s\n", s.Template, s.Filename, f.Name(), host.Client.RemoteAddr(), s.Destination)
	return scp.Copy(stat.Size(), os.ModePerm, f.Name(), f, s.Destination, sess)
}

func RewriteTemplateFile(host Host, filename string) (string, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", err
	}
	tmpfile := strconv.FormatInt(time.Now().Unix(), 10) + ".tmp"
	if err := ioutil.WriteFile(tmpfile, []byte(Template(host.Variables, string(data))), 0777); err != nil {
		return tmpfile, err
	}
	return tmpfile, err
}

func Template(vars map[string]string, value string) string {
	var i int
	var output string
	for i < len(value) {
		switch value[i] {
		case '{':
			if i < len(value)-1 && value[i+1] == '{' {
				i += 2
				var hostVar string
				i, hostVar = getVariable(vars, value, i)
				output += hostVar
			}
		default:
			output += string(value[i])
		}
		i++
	}
	return output
}

func getVariable(vars map[string]string, value string, i int) (int, string) {
	start := i
	for i < len(value) {
		switch value[i] {
		case '}':
			if i < len(value)-1 && value[i+1] == '}' {
				v := value[start:i]
				variable, ok := vars[v]
				if !ok {
					panic(fmt.Sprintf("no variable with key: %v", v))
				}
				return i+1, variable
			}
		}
		i++
	}
	return len(value), value[i:]
}
