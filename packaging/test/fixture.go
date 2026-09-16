// TEST FIXTURE ONLY. This is not the Sushiro CLI and must never be published.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		os.Exit(2)
	}
	switch os.Args[1] {
	case "echo":
		input, _ := io.ReadAll(os.Stdin)
		fmt.Fprintln(os.Stderr, "FIXTURE stderr")
		json.NewEncoder(os.Stdout).Encode(map[string]any{"fixture": true, "args": os.Args[2:], "stdin": string(input)})
	case "mcp":
		io.Copy(os.Stdout, os.Stdin)
	case "exit":
		code, _ := strconv.Atoi(os.Args[2])
		os.Exit(code)
	case "signal":
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM, syscall.SIGHUP)
		fmt.Fprintln(os.Stdout, "READY")
		<-ch
		fmt.Fprintln(os.Stderr, "FIXTURE received signal")
		os.Exit(42)
	case "self-signal":
		fmt.Fprintln(os.Stdout, "READY")
		for {
			time.Sleep(time.Hour)
		}
	default:
		os.Exit(2)
	}
}
