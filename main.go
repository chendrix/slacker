package main

import (
	"fmt"
	"github.com/chendrix/slacker/config"
	"os"
)

func main() {
	cfg, err := config.NewConfig(os.Args)
	if err != nil {
		panic(err)
	}

	err = cfg.Validate()
	if err != nil {
		panic(err)
	}

	fmt.Println(cfg)
	fmt.Println("Hello, World!")
}
