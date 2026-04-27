package main

import "github.com/lustertools/lusterpass/cmd"

var version = "dev"

func main() {
	cmd.SetVersion(version)
	cmd.Execute()
}
