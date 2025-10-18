package main

import (
	"nudgebee.com/nbctl/cmd"
	"nudgebee.com/nbctl/pkg/config"
)

func main() {
	config.InitConfig()
	cmd.Execute()
}
