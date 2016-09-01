package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"golang.org/x/net/context"

	"github.com/pkg/errors"
	"github.com/urfave/cli"

	"github.com/currantlabs/ble"
	"github.com/currantlabs/ble/examples/lib"
	"github.com/currantlabs/ble/examples/lib/dev"
	"github.com/currantlabs/ble/linux"
)

var device ble.Device

func main() {
	app := cli.NewApp()

	app.Name = "blesh"
	app.Usage = "A CLI tool for ble"
	app.Version = "0.0.1"
	app.Action = cli.ShowAppHelp
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "device",
			Value: "default",
			Usage: "implementation of ble (default / bled)",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "scan",
			Aliases: []string{"s"},
			Usage:   "Scan surrounding with specified filter",
			Action:  scan,
			Flags: []cli.Flag{
				cli.DurationFlag{Name: "duration, d", Value: time.Second * 5, Usage: "duration"},
				cli.StringFlag{Name: "name, n", Usage: "name"},
				cli.StringFlag{Name: "addr, a", Usage: "addr"},
				cli.BoolTFlag{Name: "dup", Usage: "allowDup"},
			},
		},
		{
			Name:    "exp",
			Aliases: []string{"e"},
			Usage:   "Scan and explore surrounding with specified filter",
			Action:  exp,
			Flags: []cli.Flag{
				cli.DurationFlag{Name: "duration, d", Value: time.Second * 5, Usage: "duration"},
				cli.StringFlag{Name: "name, n", Usage: "name"},
				cli.StringFlag{Name: "addr, a", Usage: "addr"},
				cli.BoolTFlag{Name: "dup", Usage: "allowDup"},
				cli.DurationFlag{Name: "sub", Usage: "subscribe to notifications and indications"},
			},
		},
		{
			Name:    "adv",
			Aliases: []string{"a"},
			Usage:   "Advertise name, UUIDs, iBeacon (TODO)",
			Action:  adv,
			Flags: []cli.Flag{
				cli.DurationFlag{Name: "duration, d", Value: time.Second * 5, Usage: "duration"},
				cli.StringFlag{Name: "name, n", Value: "Gopher", Usage: "Device Name"},
			},
		},
		{
			Name:    "serve",
			Aliases: []string{"sv"},
			Usage:   "Start the GATT Server",
			Action:  serve,
			Flags: []cli.Flag{
				cli.DurationFlag{Name: "duration, d", Value: time.Second * 5, Usage: "duration"},
				cli.StringFlag{Name: "name, n", Value: "Gopher", Usage: "Device Name"},
			},
		},
		{
			Name:    "shell",
			Aliases: []string{"sh"},
			Usage:   "Entering interactive mode",
			Action:  func(c *cli.Context) { shell(app) },
		},
	}

	app.Before = setup
	app.Run(os.Args)
}

func setup(c *cli.Context) error {
	if device != nil {
		return nil
	}
	fmt.Printf("Initializing device ...\n")
	d, err := dev.NewDevice("device")
	if err != nil {
		return errors.Wrap(err, "can't new device")
	}
	ble.SetDefaultDevice(d)
	device = d

	// Optinal. Demostrate changing HCI parameters on Linux.
	if dev, ok := d.(*linux.Device); ok {
		return errors.Wrap(updateLinuxParam(dev), "can't update hci parameters")
	}

	return nil
}

func adv(c *cli.Context) error {
	fmt.Printf("Advertising for %s...\n", c.Duration("d"))
	ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), c.Duration("d")))
	return chkErr(ble.AdvertiseNameAndServices(ctx, "Gopher"))
}

func scan(c *cli.Context) error {
	fmt.Printf("Scanning for %s...\n", c.Duration("d"))
	ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), c.Duration("d")))
	return chkErr(ble.Scan(ctx, c.Bool("dup"), advHandler, filter(c)))
}

func serve(c *cli.Context) error {
	testSvc := ble.NewService(lib.TestSvcUUID)
	testSvc.AddCharacteristic(lib.NewCountChar())
	testSvc.AddCharacteristic(lib.NewEchoChar())

	if err := ble.AddService(testSvc); err != nil {
		return errors.Wrap(err, "can't add service")
	}

	fmt.Printf("Serving GATT Server for %s...\n", c.Duration("d"))
	ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), c.Duration("d")))
	return chkErr(ble.AdvertiseNameAndServices(ctx, "Gopher", testSvc.UUID))
}

func exp(c *cli.Context) error {
	ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), c.Duration("d")))
	cln, err := ble.Connect(ctx, filter(c))
	if err != nil {
		return err
	}

	explorer(cln, c.Duration("sub"))
	fmt.Printf("Disconnecting [ %s ]... (this might take up to few seconds on OS X)\n", cln.Address())
	return cln.CancelConnection()
}

func shell(app *cli.App) {
	reader := bufio.NewReader(os.Stdin)
	sigs := make(chan os.Signal, 1)
	go func() {
		for range sigs {
			fmt.Printf("\n(type quit or q to exit)\n")
		}
	}()
	defer close(sigs)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	for {
		fmt.Print("blesh > ")
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		if text == "quit" || text == "q" {
			break
		}
		app.Run(append(os.Args[1:], strings.Split(text, " ")...))
	}
	signal.Stop(sigs)
}

func chkErr(err error) error {
	switch errors.Cause(err) {
	case context.DeadlineExceeded:
		// Sepcified duration passed, which is the expected case.
		return nil
	case context.Canceled:
		fmt.Printf("\n(Canceled)\n")
		return nil
	}
	return err
}
