package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"golang.org/x/net/context"

	"github.com/pkg/errors"
	"github.com/urfave/cli"

	"github.com/currantlabs/ble"
	"github.com/currantlabs/ble/examples/lib"
	"github.com/currantlabs/ble/examples/lib/dev"
	"github.com/currantlabs/ble/linux"
)

var curr struct {
	device  ble.Device
	client  ble.Client
	clients map[string]ble.Client
	uuid    ble.UUID
	handle  uint16
	addr    ble.Addr
	profile *ble.Profile
	val     *string
}

var (
	errNotConnected  = fmt.Errorf("not connected")
	errNoProfile     = fmt.Errorf("no profile")
	errNoUUID        = fmt.Errorf("no UUID")
	errInvalidUUID   = fmt.Errorf("invalid UUID")
	errInvalidHandle = fmt.Errorf("invalid handle")
	errNoHandle      = fmt.Errorf("no handle")
	errNoVal         = fmt.Errorf("no value")
)

func main() {
	curr.clients = make(map[string]ble.Client)

	app := cli.NewApp()

	app.Name = "blesh"
	app.Usage = "A CLI tool for ble"
	app.Version = "0.0.1"
	app.Action = cli.ShowAppHelp

	app.Commands = []cli.Command{
		{
			Name:    "status",
			Aliases: []string{"st"},
			Usage:   "Display current status",
			Before:  setup,
			Action:  cmdStatus,
		},
		{
			Name:    "adv",
			Aliases: []string{"a"},
			Usage:   "Advertise name, UUIDs, iBeacon (TODO)",
			Before:  setup,
			Action:  cmdAdv,
			Flags:   []cli.Flag{flgTimeout, flgName},
		},
		{
			Name:    "serve",
			Aliases: []string{"sv"},
			Usage:   "Start the GATT Server",
			Before:  setup,
			Action:  cmdServe,
			Flags:   []cli.Flag{flgTimeout, flgName},
		},
		{
			Name:    "scan",
			Aliases: []string{"s"},
			Usage:   "Scan surrounding with specified filter",
			Before:  setup,
			Action:  cmdScan,
			Flags:   []cli.Flag{flgTimeout, flgName, flgAddr, flgSvc, flgAllowDup},
		},
		{
			Name:    "connect",
			Aliases: []string{"c"},
			Usage:   "Connect to a peripheral device",
			Before:  setup,
			Action:  cmdConnect,
			Flags:   []cli.Flag{flgTimeout, flgName, flgAddr, flgSvc},
		},
		{
			Name:    "disconnect",
			Aliases: []string{"x"},
			Usage:   "Disconnect a connected peripheral device",
			Before:  setup,
			Action:  cmdDisconnect,
			Flags:   []cli.Flag{flgAddr},
		},
		{
			Name:    "discover",
			Aliases: []string{"d"},
			Usage:   "Discover profile on connected device",
			Before:  setup,
			Action:  cmdDiscover,
			Flags:   []cli.Flag{flgTimeout, flgName, flgAddr},
		},
		{
			Name:    "explore",
			Aliases: []string{"e"},
			Usage:   "Display discovered profile",
			Before:  setup,
			Action:  cmdExplore,
			Flags:   []cli.Flag{flgTimeout, flgName, flgAddr},
		},
		{
			Name:    "read",
			Aliases: []string{"r"},
			Usage:   "Read value from a characteristic or descriptor",
			Before:  setup,
			Action:  cmdRead,
			Flags:   []cli.Flag{flgUUID, flgHandle, flgTimeout, flgName, flgAddr},
		},
		{
			Name:    "write",
			Aliases: []string{"w"},
			Usage:   "Write value to a characteristic or descriptor",
			Before:  setup,
			Action:  cmdWrite,
			Flags:   []cli.Flag{flgUUID, flgHandle, flgVal, flgTimeout, flgName, flgAddr},
		},
		{
			Name:   "sub",
			Usage:  "Subscribe to notification (or indication)",
			Before: setup,
			Action: cmdSub,
			Flags:  []cli.Flag{flgUUID, flgInd, flgTimeout, flgName, flgAddr},
		},
		{
			Name:   "unsub",
			Usage:  "Unsubscribe to notification (or indication)",
			Before: setup,
			Action: cmdUnsub,
			Flags:  []cli.Flag{flgUUID, flgInd, flgAddr},
		},
		{
			Name:    "shell",
			Aliases: []string{"sh"},
			Usage:   "Enter interactive mode",
			Before:  setup,
			Action:  func(c *cli.Context) { cmdShell(app) },
		},
	}

	// app.Before = setup
	app.Run(os.Args)
}

func setup(c *cli.Context) error {
	if curr.device != nil {
		return nil
	}
	fmt.Printf("Initializing device ...\n")
	d, err := dev.NewDevice("default")
	if err != nil {
		return errors.Wrap(err, "can't new device")
	}
	ble.SetDefaultDevice(d)
	curr.device = d

	// Optinal. Demostrate changing HCI parameters on Linux.
	if dev, ok := d.(*linux.Device); ok {
		return errors.Wrap(updateLinuxParam(dev), "can't update hci parameters")
	}

	return nil
}
func cmdStatus(c *cli.Context) error {
	m := map[bool]string{true: "yes", false: "no"}
	fmt.Printf("Current status:\n")
	fmt.Printf("  Initialized: %s\n", m[curr.device != nil])

	if curr.addr != nil {
		fmt.Printf("  Address:     %s\n", curr.addr)
	} else {
		fmt.Printf("  Address:\n")
	}

	fmt.Printf("  Connected:")
	for k := range curr.clients {
		fmt.Printf(" %s", k)
	}
	fmt.Printf("\n")

	fmt.Printf("  Profile:\n")
	if curr.profile != nil {
		fmt.Printf("\n")
		explore(curr.client, curr.profile)
	}

	if curr.uuid != nil {
		fmt.Printf("  UUID:       %s\n", curr.uuid)
	} else {
		fmt.Printf("  UUID:\n")
	}

	return nil
}

func cmdAdv(c *cli.Context) error {
	fmt.Printf("Advertising for %s...\n", c.Duration("tmo"))
	ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), c.Duration("tmo")))
	return chkErr(ble.AdvertiseNameAndServices(ctx, "Gopher"))
}

func cmdScan(c *cli.Context) error {
	fmt.Printf("Scanning for %s...\n", c.Duration("tmo"))
	ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), c.Duration("tmo")))
	return chkErr(ble.Scan(ctx, c.Bool("dup"), advHandler, filter(c)))
}

func cmdServe(c *cli.Context) error {
	testSvc := ble.NewService(lib.TestSvcUUID)
	testSvc.AddCharacteristic(lib.NewCountChar())
	testSvc.AddCharacteristic(lib.NewEchoChar())

	if err := ble.AddService(testSvc); err != nil {
		return errors.Wrap(err, "can't add service")
	}

	fmt.Printf("Serving GATT Server for %s...\n", c.Duration("tmo"))
	ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), c.Duration("tmo")))
	return chkErr(ble.AdvertiseNameAndServices(ctx, "Gopher", testSvc.UUID))
}

func cmdConnect(c *cli.Context) error {
	curr.client = nil

	var cln ble.Client
	var err error

	ctx := ble.WithSigHandler(context.WithTimeout(context.Background(), c.Duration("tmo")))
	if c.String("addr") != "" {
		curr.addr = ble.NewAddr(c.String("addr"))
		fmt.Printf("Dialing to specified address: %s\n", curr.addr)
		cln, err = ble.Dial(ctx, curr.addr)
	} else if filter(c) != nil {
		fmt.Printf("Scanning with filter...\n")
		if cln, err = ble.Connect(ctx, filter(c)); err == nil {
			curr.addr = cln.Address()
			fmt.Printf("Connected to %s\n", curr.addr)

		}
	} else if curr.addr != nil {
		fmt.Printf("Dialing to implicit address: %s\n", curr.addr)
		cln, err = ble.Dial(ctx, curr.addr)
	} else {
		return fmt.Errorf("no filter specified, and cached peripheral address")
	}
	if err == nil {
		curr.client = cln
		curr.clients[cln.Address().String()] = cln
		go func() {
			<-cln.Disconnected()
			delete(curr.clients, cln.Address().String())
			curr.client = nil
			fmt.Printf("\n%s disconnected\n", cln.Address().String())
		}()
	}
	return err
}

func cmdDisconnect(c *cli.Context) error {
	if c.String("addr") != "" {
		curr.client = curr.clients[c.String("addr")]
	}
	if curr.client == nil {
		return errNotConnected
	}
	defer func() {
		delete(curr.clients, curr.client.Address().String())
		curr.client = nil
		curr.profile = nil
	}()

	fmt.Printf("Disconnecting [ %s ]... (this might take up to few seconds on OS X)\n", curr.client.Address())
	return curr.client.CancelConnection()
}

func cmdDiscover(c *cli.Context) error {
	curr.profile = nil
	if curr.client == nil {
		if err := cmdConnect(c); err != nil {
			return errors.Wrap(err, "can't connect")
		}
	}

	fmt.Printf("Discovering profile...\n")
	p, err := curr.client.DiscoverProfile(true)
	if err != nil {
		return errors.Wrap(err, "can't discover profile")
	}

	curr.profile = p
	return nil
}

func cmdExplore(c *cli.Context) error {
	if curr.client == nil {
		if err := cmdConnect(c); err != nil {
			return errors.Wrap(err, "can't connect")
		}
	}
	if curr.profile == nil {
		if err := cmdDiscover(c); err != nil {
			return errors.Wrap(err, "can't discover profile")
		}
	}
	return explore(curr.client, curr.profile)
}

func cmdRead(c *cli.Context) error {
	curr.uuid = nil
	if err := doGetUUID(c); err != nil {
		if err.Error() == errNoUUID.Error() {
			if err := doGetHandle(c); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	if err := doConnect(c); err != nil {
		return err
	}
	if err := doDiscover(c); err != nil {
		return err
	}
	if curr.uuid != nil {
		if u := curr.profile.Find(ble.NewCharacteristic(curr.uuid)); u != nil {
			b, err := curr.client.ReadCharacteristic(u.(*ble.Characteristic))
			if err != nil {
				return errors.Wrap(err, "can't read characteristic")
			}
			fmt.Printf("    Value         %x | %q\n", b, b)
			return nil
		}
		if u := curr.profile.Find(ble.NewDescriptor(curr.uuid)); u != nil {
			b, err := curr.client.ReadDescriptor(u.(*ble.Descriptor))
			if err != nil {
				return errors.Wrap(err, "can't read descriptor")
			}
			fmt.Printf("    Value         %x | %q\n", b, b)
			return nil
		}
	} else {
		dummyCharacteristic := ble.NewCharacteristic(ble.UUID16(curr.handle))
		dummyCharacteristic.Handle = curr.handle
		if u := curr.profile.FindByHandle(dummyCharacteristic); u != nil {
			b, err := curr.client.ReadCharacteristic(u.(*ble.Characteristic))
			if err != nil {
				return errors.Wrap(err, "can't read characteristic")
			}
			fmt.Printf("    Value         %x | %q\n", b, b)
			return nil
		}
		dummyDescriptor := ble.NewDescriptor(ble.UUID16(curr.handle))
		dummyDescriptor.Handle = curr.handle
		if u := curr.profile.FindByHandle(dummyDescriptor); u != nil {
			b, err := curr.client.ReadDescriptor(u.(*ble.Descriptor))
			if err != nil {
				return errors.Wrap(err, "can't read descriptor")
			}
			fmt.Printf("    Value         %x | %q\n", b, b)
			return nil
		}
	}

	return errNoUUID
}

func cmdWrite(c *cli.Context) error {
	curr.uuid = nil
	curr.val = nil
	if err := doGetUUID(c); err != nil {
		if err.Error() == errNoUUID.Error() {
			if err := doGetHandle(c); err != nil {
				return err
			}
		} else {
			return err
		}
	}
	if err := doGetVal(c); err != nil {
		return err
	}
	if err := doConnect(c); err != nil {
		return err
	}
	if err := doDiscover(c); err != nil {
		return err
	}
	if curr.uuid != nil {
		if u := curr.profile.Find(ble.NewCharacteristic(curr.uuid)); u != nil {
			err := curr.client.WriteCharacteristic(u.(*ble.Characteristic), []byte(*curr.val), true)
			return errors.Wrap(err, "can't write characteristic")
		}
		if u := curr.profile.Find(ble.NewDescriptor(curr.uuid)); u != nil {
			err := curr.client.WriteDescriptor(u.(*ble.Descriptor), []byte(*curr.val))
			return errors.Wrap(err, "can't write descriptor")
		}
	} else {
		dummyCharacteristic := ble.NewCharacteristic(ble.UUID16(curr.handle))
		dummyCharacteristic.Handle = curr.handle
		if u := curr.profile.FindByHandle(dummyCharacteristic); u != nil {
			err := curr.client.WriteCharacteristic(u.(*ble.Characteristic), []byte(*curr.val), true)
			return errors.Wrap(err, "can't write characteristic")
		}
		dummyDescriptor := ble.NewDescriptor(ble.UUID16(curr.handle))
		dummyDescriptor.Handle = curr.handle
		if u := curr.profile.FindByHandle(dummyDescriptor); u != nil {
			err := curr.client.WriteDescriptor(u.(*ble.Descriptor), []byte(*curr.val))
			return errors.Wrap(err, "can't write descriptor")
		}
	}
	return errNoUUID
}

func cmdSub(c *cli.Context) error {
	if err := doGetUUID(c); err != nil {
		return err
	}
	if err := doConnect(c); err != nil {
		return err
	}
	if err := doDiscover(c); err != nil {
		return err
	}
	// NotificationHandler
	h := func(req []byte) { fmt.Printf("notified: %x | %q\n", req, req) }
	if u := curr.profile.Find(ble.NewCharacteristic(curr.uuid)); u != nil {
		err := curr.client.Subscribe(u.(*ble.Characteristic), c.Bool("ind"), h)
		return errors.Wrap(err, "can't subscribe to characteristic")
	}
	return errNoUUID
}

func cmdUnsub(c *cli.Context) error {
	if err := doGetUUID(c); err != nil {
		return err
	}
	if err := doConnect(c); err != nil {
		return err
	}
	if u := curr.profile.Find(ble.NewCharacteristic(curr.uuid)); u != nil {
		err := curr.client.Unsubscribe(u.(*ble.Characteristic), c.Bool("ind"))
		return errors.Wrap(err, "can't unsubscribe to characteristic")
	}
	return errNoUUID
}

func cmdShell(app *cli.App) {
	cli.OsExiter = func(c int) {}
	reader := bufio.NewReader(os.Stdin)
	sigs := make(chan os.Signal, 1)
	go func() {
		for range sigs {
			fmt.Printf("\n(type quit or q to exit)\n\nblesh >")
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
		innerArgs, err := parseCommandLine(text)
		if err != nil {
			fmt.Printf("\nPre-parser error: %s", err.Error())
			break
		}
		app.Run(append(os.Args[1:], innerArgs...))
	}
	signal.Stop(sigs)
}

// Correctly takes into consideration spaces passed inside strings
// (e.g. when using --value="this has spaces" to write to characteristics / descriptors).
// Originally from:
// https://stackoverflow.com/questions/34118732/parse-a-command-line-string-into-flags-and-arguments-in-golang
// Adapted and a space-bug fix, for the first space after the first character.
func parseCommandLine(command string) ([]string, error) {
	var args []string
	state := "start"
	current := ""
	quote := "\""
	escapeNext := true
	for i := 0; i < len(command); i++ {
		c := command[i]

		if state == "quotes" {
			if string(c) != quote {
				current += string(c)
			} else {
				args = append(args, current)
				current = ""
				state = "start"
			}
			continue
		}

		if escapeNext {
			current += string(c)
			escapeNext = false
			continue
		}

		if c == '\\' {
			escapeNext = true
			continue
		}

		if c == '"' || c == '\'' {
			state = "quotes"
			quote = string(c)
			continue
		}

		if state == "arg" {
			if c == ' ' || c == '\t' {
				args = append(args, current)
				current = ""
				state = "start"
			} else {
				current += string(c)
			}
			continue
		}

		if c != ' ' && c != '\t' {
			state = "arg"
			current += string(c)
			continue
		}

		if c == ' ' || c == '\t' && current != "" {
			args = append(args, current)
			current = ""
			state = "start"
		}
	}

	if state == "quotes" {
		return []string{}, errors.New(fmt.Sprintf("Unclosed quote in command line: %s", command))
	}

	if current != "" {
		args = append(args, current)
	}

	return args, nil
}
