package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"

	"github.com/air-verse/air/runner"
)

var (
	cfgPath     string
	debugMode   bool
	showVersion bool
	cmdArgs     map[string]runner.TomlInfo
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
	parseFlag(os.Args[1:])
}

func parseFlag(args []string) {
	flag.Usage = helpMessage
	flag.StringVar(&cfgPath, "c", "", "config path")
	flag.BoolVar(&debugMode, "d", false, "debug mode")
	flag.BoolVar(&showVersion, "v", false, "show version")
	cmd := flag.CommandLine
	cmdArgs = runner.ParseConfigFlag(cmd)
	if err := flag.CommandLine.Parse(args); err != nil {
		log.Fatal(err)
	}
}

type versionInfo struct {
	airVersion string
	goVersion  string
}

func GetVersionInfo() versionInfo { //revive:disable:unexported-return
	if len(airVersion) != 0 && len(goVersion) != 0 {
		return versionInfo{
			airVersion: airVersion,
			goVersion:  goVersion,
		}
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		return versionInfo{
			airVersion: info.Main.Version,
			goVersion:  runtime.Version(),
		}
	}
	return versionInfo{
		airVersion: "(unknown)",
		goVersion:  runtime.Version(),
	}
}

func printSplash() {
	versionInfo := GetVersionInfo()
	fmt.Printf(`
  __    _   ___  
 / /\  | | | |_) 
/_/--\ |_| |_| \_ %s, built with Go %s

`, versionInfo.airVersion, versionInfo.goVersion)
}

func main() {
	if showVersion {
		printSplash()
		return
	}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	var err error
	cfg, err := runner.InitConfig(cfgPath)
	if err != nil {
		log.Fatal(err)
		return
	}
	cfg.WithArgs(cmdArgs)
	if !cfg.Log.Silent {
		printSplash()
	}
	if debugMode && !cfg.Log.Silent {
		fmt.Println("[debug] mode")
	}
	r, err := runner.NewEngineWithConfig(cfg, debugMode)
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
