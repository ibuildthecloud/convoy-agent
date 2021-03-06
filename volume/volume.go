package volume

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"

	convoyflags "github.com/rancher/convoy/client/flags"

	"github.com/rancher/convoy-agent/cattle"
)

const convoyFlagNamePrefix string = "convoy-"
const convoyFlagUsagePrefix string = "Passed to convoy. "
const flagFmt string = "--%s=%s"

var convoyFlagNames = []string{}
var convoyFlags = map[string]string{}
var rootUuidFileName = "UUID"
var Commands = []cli.Command{
	{
		Name:      "volume",
		Usage:     "Start convoy-agent as a volume agent",
		Action:    volumeAgent,
		ShortName: "v",
	},
}

func init() {
	flags := []cli.Flag{
		cli.StringFlag{
			Name:  "socket, s",
			Value: "/var/run/convoy/convoy.sock",
			Usage: "specify unix domain socket for communicating with convoy server",
		},
		cli.StringFlag{
			Name:   "host-uuid",
			Usage:  "set the host uuid for the host",
			EnvVar: "CATTLE_HOST_UUID",
		},
	}

	for _, f := range convoyflags.DaemonFlags {
		// This type switch is annoying, but Name is not exposed on the cli.Flag struct, so we have to cast to the specific types.
		switch typedF := f.(type) {
		case cli.StringFlag:
			typedF.Name = convoyFlagNamePrefix + typedF.Name
			typedF.Usage = convoyFlagUsagePrefix + typedF.Usage
			flags = append(flags, typedF)
			convoyFlags[typedF.Name] = "string"
			convoyFlagNames = append(convoyFlagNames, typedF.Name)
		case cli.StringSliceFlag:
			typedF.Name = convoyFlagNamePrefix + typedF.Name
			typedF.Usage = convoyFlagUsagePrefix + typedF.Usage
			flags = append(flags, typedF)
			convoyFlags[typedF.Name] = "stringslice"
			convoyFlagNames = append(convoyFlagNames, typedF.Name)
		case cli.BoolFlag:
			typedF.Name = convoyFlagNamePrefix + typedF.Name
			typedF.Usage = convoyFlagUsagePrefix + typedF.Usage
			flags = append(flags, typedF)
			convoyFlags[typedF.Name] = "bool"
			convoyFlagNames = append(convoyFlagNames, typedF.Name)
		default:
			logrus.Fatalf("Unknown type. Can't use convoy flag: %#v", f)
		}
	}
	Commands[0].Flags = flags
}

func volumeAgent(c *cli.Context) {
	socket := c.String("socket")
	cattleUrl := c.GlobalString("url")
	cattleAccessKey := c.GlobalString("access-key")
	cattleSecretKey := c.GlobalString("secret-key")
	if c.GlobalBool("debug") {
		logrus.SetLevel(logrus.DebugLevel)
	}

	healthCheckInterval := c.GlobalInt("healthcheck-interval")
	healthCheckBaseDir := c.GlobalString("healthcheck-basedir")

	driver := c.GlobalString("storagepool-driver")
	if driver == "" {
		logrus.Fatal("required field storagepool-driver has not been set")
	}

	hostUuid := c.String("host-uuid")
	if hostUuid == "" {
		logrus.Fatal("required field host-uuid has not been set")
	}

	resultChan := make(chan error)

	go func(rc chan<- error) {
		cmdArgs := buildConvoyCmdArgs(c, socket)
		cmd := exec.Command("convoy", cmdArgs...)
		logrus.Infof("Launching convoy with args: %s", cmdArgs)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		logrus.Infof("convoy exited with error: %v", err)
		rc <- err
	}(resultChan)

	go func(rc chan<- error) {
		controlChan := make(chan bool, 1)
		cattleClient, err := cattle.NewCattleClient(cattleUrl, cattleAccessKey, cattleSecretKey)
		if err != nil {
			rc <- fmt.Errorf("Error getting cattle client: %v", err)
		}
		volAgent := NewVolumeAgent(healthCheckBaseDir, socket, hostUuid, healthCheckInterval, cattleClient, driver)
		err = volAgent.Run(controlChan)
		logrus.Infof("volume-agent exited with error: %v", err)
		rc <- err
	}(resultChan)

	<-resultChan
	logrus.Info("Exiting.")
}

func buildConvoyCmdArgs(c *cli.Context, socket string) []string {
	convoyCmd := []string{fmt.Sprintf(flagFmt, "socket", socket), "daemon"}
	for flagName, flagType := range convoyFlags {
		if !c.IsSet(flagName) {
			continue
		}
		f := c.Generic(flagName)
		flagName = flagName[len(convoyFlagNamePrefix):]
		logrus.Infof("Got: %s %v", flagName, f)
		switch flagType {
		case "string":
			fallthrough
		case "bool":
			fl := f.(flag.Getter)
			convoyCmd = append(convoyCmd, fmt.Sprintf(flagFmt, flagName, fl.String()))
		case "stringslice":
			fl := f.(*cli.StringSlice)
			for _, val := range fl.Value() {
				convoyCmd = append(convoyCmd, fmt.Sprintf(flagFmt, flagName, val))
			}
		}
	}
	return convoyCmd
}
