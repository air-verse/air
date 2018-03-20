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
var debugMode bool

func init() {
	flag.StringVar(&cfgPath, "c", "", "config path")
	flag.BoolVar(&debugMode, "d", false, "debug mode")
	flag.Parse()
}

func main() {
	fmt.Printf(`
U  /"\  u       ___      U |  _"\ u  
 \/ _ \/       |_"_|      \| |_) |/  
 / ___ \        | |        |  _ <    
/_/   \_\     U/| |\u      |_| \_\   
 \\    >>  .-,_|___|_,-.   //   \\_  
(__)  (__)  \_)-' '-(_/   (__)  (__)

~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
  Live reload for Go apps - v%s

`, version)
	if debugMode {
		fmt.Println("[debug] mode")
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	var err error
	r, err := runner.NewEngine(cfgPath, debugMode)
	if err != nil {
		log.Fatal(err)
	}
	go func() {
		<-sigs
		r.Stop()
	}()

	defer func() {
		if e := recover(); e != nil {
			log.Fatalf("PANIC: %+v", e)
		}
	}()

	r.Run()
}
