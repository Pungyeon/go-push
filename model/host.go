package model

import "golang.org/x/crypto/ssh"

type Host struct {
	Address string `yaml:"address"`
	Port string `yaml:"port"`
	Password string `yaml:"password"`
	Variables map[string]string
	Client *ssh.Client
}

func (host Host) WithClient(client *ssh.Client) Host {
	host.Client = client
	return host
}