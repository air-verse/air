package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/cosmtrek/air/runner"
)

var cfgPath string

func main() {
	fmt.Print(`
             _
     /\     (_)
    /  \     _   _ __
   / /\ \   | | | '__|
  / ____ \  | | | |
 /_/    \_\ |_| |_|

Live reload for Go apps :)

`)
	flag.StringVar(&cfgPath, "c", "", "config path")
	flag.Parse()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	var err error
	r, err := runner.NewEngine(cfgPath)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		<-sigs
		r.Stop()
	}()

	r.Run()
}
