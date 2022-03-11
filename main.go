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

var (
	cfgPath     string
	debugMode   bool
	showVersion bool
)

func helpMessage() {
	fmt.Fprintf(flag.CommandLine.Output(), "Usage of %s:\n\n", os.Args[0])
	fmt.Printf("If no command is provided %s will start the runner with the provided flags\n\n", os.Args[0])
	fmt.Println("Commands:")
	fmt.Print("  init	creates a .air.toml file with default settings to the current directory\n\n")

	fmt.Println("Flags:")
	flag.PrintDefaults()
}

func init() {
	flag.Usage = helpMessage
	flag.StringVar(&cfgPath, "c", "", "config path")
	flag.BoolVar(&debugMode, "d", false, "debug mode")
	flag.BoolVar(&showVersion, "v", false, "show version")
	flag.Parse()
}

func main() {
	fmt.Printf(`
  __    _   ___  
 / /\  | | | |_) 
/_/--\ |_| |_| \_ %s, built with Go %s

`, airVersion, goVersion)

	if showVersion {
		return
	}

	if debugMode {
		fmt.Println("[debug] mode")
	}

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	var err error
	r, err := runner.NewEngine(cfgPath, debugMode)
	if err != nil {
		log.Fatal(err)
		return
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
