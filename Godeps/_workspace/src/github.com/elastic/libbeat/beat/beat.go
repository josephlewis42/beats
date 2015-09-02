package beat

import (
	"flag"
	"fmt"
	"github.com/elastic/libbeat/cfgfile"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/outputs"
	"github.com/elastic/libbeat/publisher"
	"github.com/elastic/libbeat/service"
	"os"
	"runtime"
)

// Beater interface that every beat must use
type Beater interface {
	Config(*Beat) error
	Setup(*Beat) error
	Run(*Beat) error
	Cleanup(*Beat) error
	Stop()
}

// Basic beat information
type Beat struct {
	Name    string
	Version string
	CmdLine *flag.FlagSet
	Config  BeatConfig
	BT      Beater
}

// Basic configuration of every beat
type BeatConfig struct {
	Output  map[string]outputs.MothershipConfig
	Logging logp.Logging
	Shipper publisher.ShipperConfig
}

// Initiates a new beat object
func NewBeat(name string, version string, bt Beater) *Beat {
	b := Beat{
		Version: version,
		Name:    name,
		BT:      bt,
	}

	b.CmdLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	return &b
}

// Reads and parses the default command line params
// To set additional cmd line args use the beat.CmdLine type before calling the function
func (beat *Beat) CommandLineSetup() {

	cfgfile.CmdLineFlags(beat.CmdLine, beat.Name)
	logp.CmdLineFlags(beat.CmdLine)
	service.CmdLineFlags(beat.CmdLine)
	publisher.CmdLineFlags(beat.CmdLine)

	printVersion := beat.CmdLine.Bool("version", false, "Print version and exit")

	beat.CmdLine.Parse(os.Args[1:])

	if *printVersion {
		fmt.Printf("%s version %s (%s)\n", beat.Name, beat.Version, runtime.GOARCH)
		return
	}
}

// Inits the config file and reads the default config information into Beat.Config
// This is Output, Logging and Shipper config params
func (b *Beat) LoadConfig() {

	err := cfgfile.Read(&b.Config)

	if err != nil {
		logp.Debug("Log read error", "Error %v\n", err)
	}

	logp.Init(b.Name, &b.Config.Logging)

	logp.Debug("main", "Initializing output plugins")

	if err := publisher.Publisher.Init(b.Name, b.Config.Output, b.Config.Shipper); err != nil {
		logp.Critical(err.Error())
		os.Exit(1)
	}

	logp.Debug(b.Name, "Init %s", b.Name)
}

// internal libbeat function that calls beater Run method
func (b *Beat) Run() {

	// Setup beater object
	b.BT.Setup(b)

	// Up to here was the initialization, now about running
	if cfgfile.IsTestConfig() {
		// all good, exit with 0
		os.Exit(0)
	}
	service.BeforeRun()

	// Run beater specific stuff
	b.BT.Run(b)

	// Function called in case of beater stop
	service.HandleSignals(b.BT.Stop)
	service.Cleanup()

	logp.Debug("main", "Cleanup")

	// Call beater cleanup function
	b.BT.Cleanup(b)
}

// Stops the beat and calls the beater Stop action
func (beat *Beat) Stop() {
	beat.BT.Stop()
}
