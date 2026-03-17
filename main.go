package main

import (
	"context"
	"data-server/cmd"
)

func main() {
	context := context.Background()
	cmd.Execute(context)
}
