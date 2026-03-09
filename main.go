package main

import "github.com/niladribose/obeya/cmd"

var (
	version = "dev"
	commit  = "none"
)

func main() {
	cmd.SetVersionInfo(version, commit)
	cmd.Execute()
}
