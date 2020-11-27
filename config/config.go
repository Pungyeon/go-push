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
	Variables map[string]string `yaml:"variables"`
}

func (host *HostConfig) GetUsername(global GlobalConfig) string {
	if host.Username == "" {
		return global.Username
	}
	return host.Username
}

func (host *HostConfig) GetPassword(global GlobalConfig) string {
	if host.Username == "" {
		return global.Password
	}
	return host.Password
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
	CmdTypeBash   CmdType = "bash"
	CmdTypeUpload CmdType = "upload"
	CmdTypeDownload CmdType = "download"
)

type GlobalConfig struct {
	Async bool `yaml:"async"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Variables map[string]string `yaml:"variables"`
}

type Config struct {
	Global GlobalConfig `yaml:"global"`
	Hosts []HostConfig `yaml:"hosts"`
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
		if cmd.Type == CmdTypeUpload {
			var scp model.Upload
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
	for _, hostConfig := range c.Hosts {
		host, config := c.ParseClient(hostConfig)
		client, err := ssh.Dial("tcp", hostConfig.GetAddr(), config)
		if err != nil {
			return nil, err
		}
		clients = append(clients, host.WithClient(client))
	}
	return clients, nil
}

func (c *Config) ParseClient(host HostConfig) (model.Host, *ssh.ClientConfig) {
	return model.Host{
		Address: host.GetAddr(),
		Password: host.GetPassword(c.Global),
		Variables: addGlobalVariables(host.Variables, c.Global.Variables),
	},
	&ssh.ClientConfig{
		User: host.GetUsername(c.Global),
		Auth: []ssh.AuthMethod{
			ssh.Password(host.GetPassword(c.Global)),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}

func addGlobalVariables(vars, global map[string]string) map[string]string {
	for k, v := range global {
		if _, ok := vars[k]; !ok {
			vars[k] = v
		}
	}
	return vars
}
