package config

import (
	"github.com/mitchellh/mapstructure"
	"go-push/model"
	"golang.org/x/crypto/ssh"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type HostConfig struct {
	Address string `yaml:"address"`
	Port string `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	CertPath string  `yaml:"cert_path"`
}

func (host *HostConfig) GetAddr() string {
	if host.Port == "" {
		host.Port = "22"
	}
	return host.Address + ":" + host.Port
}

type CommandConfig struct {
	Type CmdType `yaml:"type"`
	Value interface{} `yaml:"value"`
}

type CmdType string

const (
	CmdTypeBash CmdType = "bash"
	CmdTypeSCP CmdType = "scp"
)

type Config struct {
	Hosts []HostConfig
	Commands []CommandConfig `yaml:"commands"`
}

func New(paths ...string) ([]Config, error) {
	var cfgs []Config
	for _, path := range paths {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}

		var cfg Config
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return nil, err
		}
		cfgs = append(cfgs, cfg)
	}
	return cfgs, nil
}

func (c *Config) ParseCommands() ([]model.Command, error) {
	var commands []model.Command
	for _, cmd := range c.Commands {
		if cmd.Type == CmdTypeSCP {
			var scp model.SCP
			if err := mapstructure.Decode(cmd.Value, &scp); err != nil {
				return nil, err
			}
			commands = append(commands, scp)
		} else {
			var bash model.Bash
			if err := mapstructure.Decode(cmd.Value, &bash); err != nil {
				return nil, err
			}
			commands = append(commands, bash)
		}
	}
	return commands, nil
}

func (c *Config) ParseClients() ([]model.Host, error) {
	var clients []model.Host
	for _, host := range c.Hosts {
		sshConfig := &ssh.ClientConfig{
			User: host.Username,
			Auth: []ssh.AuthMethod{
				ssh.Password(host.Password),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		}

		client, err := ssh.Dial("tcp", host.GetAddr(), sshConfig)
		if err != nil {
			return nil, err
		}
		clients = append(clients, model.Host{
			Address: host.GetAddr(),
			Password: host.Password,
			Client: client,
		})
	}
	return clients, nil
}
