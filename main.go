package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"strings"
	"syscall"

	"github.com/air-verse/air/runner"
	"github.com/fatih/color"
)

var (
	cfgPath     string
	debugMode   bool
	showVersion bool
	colorMode   string
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
	flag.StringVar(&colorMode, "color", "auto", "colored output: auto, always, never")
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

func defaultSplashText() string {
	versionInfo := GetVersionInfo()
	return fmt.Sprintf(`
  __    _   ___  
 / /\  | | | |_) 
/_/--\ |_| |_| \_ %s, built with Go %s

`, versionInfo.airVersion, versionInfo.goVersion)
}

func versionLineText() string {
	versionInfo := GetVersionInfo()
	return fmt.Sprintf("air %s, built with Go %s\n", versionInfo.airVersion, versionInfo.goVersion)
}

func startupBannerText(cfg *runner.Config) string {
	if cfg == nil || cfg.Misc.StartupBanner == nil {
		return defaultSplashText()
	}
	return *cfg.Misc.StartupBanner
}

func printStartupBanner(cfg *runner.Config, respectSilent bool) {
	if cfg != nil && respectSilent && cfg.Log.Silent {
		return
	}
	banner := startupBannerText(cfg)
	if banner == "" {
		return
	}
	fmt.Fprint(os.Stderr, banner)
	if !strings.HasSuffix(banner, "\n") {
		fmt.Fprintln(os.Stderr)
	}
}

func printVersionOutput(cfg *runner.Config) {
	if cfg == nil || cfg.Misc.StartupBanner == nil {
		fmt.Fprint(os.Stderr, defaultSplashText())
		return
	}

	banner := *cfg.Misc.StartupBanner
	if banner != "" {
		fmt.Fprint(os.Stderr, banner)
		if !strings.HasSuffix(banner, "\n") {
			fmt.Fprintln(os.Stderr)
		}
	}

	fmt.Fprint(os.Stderr, versionLineText())
}

func main() {
	switch colorMode {
	case "always":
		color.NoColor = false
	case "never":
		color.NoColor = true
	case "auto", "":
		// do nothing
	default:
		log.Fatal("unsupported color mode: use always, never, auto")
	}

	if showVersion {
		cfg, err := runner.InitConfigForDisplay(cfgPath, cmdArgs)
		if err == nil {
			printVersionOutput(cfg)
			return
		}
		fmt.Fprint(os.Stderr, defaultSplashText())
		return
	}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	cfg, err := runner.InitConfig(cfgPath, cmdArgs)
	if err != nil {
		log.Fatal(err)
		return
	}
	printStartupBanner(cfg, true)
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
