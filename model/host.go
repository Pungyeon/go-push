package model

import "golang.org/x/crypto/ssh"

type Host struct {
	Address string `yaml:"address"`
	Port string `yaml:"port"`
	Password string `yaml:"password"`
	Client *ssh.Client
}
