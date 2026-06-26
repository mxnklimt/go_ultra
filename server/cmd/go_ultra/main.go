package main

import "fmt"

// Version 是构建版本号，后续可由 -ldflags 注入覆盖。
var Version = "0.1.0-dev"

func main() {
	fmt.Printf("go_ultra %s\n", Version)
}
