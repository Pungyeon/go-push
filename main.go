package main

import (
	"fmt"
	"github.com/spf13/pflag"
	"go-push/config"
	"time"
)

func main() {
	configs := pflag.StringSlice("configs", nil, "variadic number of configuration files")
	pflag.Parse()

	cfgs, err := config.New(*configs...)
	if err != nil {
		panic(err)
	}

	for _, cfg := range cfgs {
		hosts, err := cfg.ParseClients()
		if err != nil {
			panic(err)
		}
		commands, err := cfg.ParseCommands()
		if err != nil {
			panic(err)
		}

		for _, host := range hosts {
			fmt.Println("[*] Running Commands against:", host.Client.RemoteAddr(), host.Variables)
			for _, cmd := range commands {
				if err := cmd.Run(host); err != nil {
					time.Sleep(time.Second)
					panic(err)
				}
			}
			fmt.Println("[*] Finished running commands for:", host.Client.RemoteAddr())
			if err := host.Client.Close(); err != nil {
				fmt.Println(err)
			}
		}
		time.Sleep(time.Second)
	}
}
