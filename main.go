package main

import (
	"fmt"
	"github.com/spf13/pflag"
	"go-push/config"
	"go-push/model"
	"sync"
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

		wg := &sync.WaitGroup{}
		if cfg.Global.Async {
			RunAsync(wg, hosts, commands)
		} else {
			RunSync(wg, hosts, commands)
		}
		wg.Wait()
		time.Sleep(time.Second)
	}
}

func nope(host model.Host, err error) {
	panic(fmt.Sprintf(`Error returned:
Host: %s
Error: %v
`, host.Address, err))
}

func RunAsync(wg *sync.WaitGroup, hosts []model.Host, commands []model.Command)  {
	for _, host := range hosts {
		wg.Add(1)
		go func(host model.Host) {
			RunCommandsOnHost(wg, host, commands)
		}(host)
	}
}

func RunCommandsOnHost(wg *sync.WaitGroup, host model.Host, commands []model.Command) {
	fmt.Println("[*] Running Commands against:", host.Client.RemoteAddr(), host.Variables)
	for _, cmd := range commands {
		if err := cmd.Run(host); err != nil {
			time.Sleep(time.Second)
			nope(host, err)
		}
	}
	fmt.Println("[*] Finished running commands for:", host.Client.RemoteAddr())
	if err := host.Client.Close(); err != nil {
		fmt.Println(err)
	}
	wg.Done()
}

func RunSync(wg *sync.WaitGroup, hosts []model.Host, commands []model.Command) {
	for _, host := range hosts {
		wg.Add(1)
		RunCommandsOnHost(wg, host, commands)
	}
}