package main

import (
	"fmt"
	"strconv"

	"github.com/currantlabs/ble"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
	"golang.org/x/net/context"
)

func doGetUUID(c *cli.Context) error {
	if c.String("uuid") != "" {
		u, err := ble.Parse(c.String("uuid"))
		if err != nil {
			return errInvalidUUID
		}
		curr.uuid = u
	}
	if curr.uuid == nil {
		return errNoUUID
	}
	return nil
}

func doGetHandle(c *cli.Context) error {
	// The UUID always has priority. If already non-nil, exit
	if curr.uuid != nil {
		return nil
	}
	handleRetrieved := false
	if c.String("handle") != "" {
		hStr := c.String("handle")
		if len(hStr) > 2 && hStr[0] == '0' && (hStr[1] == 'x' || hStr[1] == 'X') {
			hStr = hStr[2:]
		}
		h, err := strconv.ParseUint(hStr, 16, 16)
		if err != nil {
			return errInvalidHandle
		}
		curr.handle = uint16(h)
		handleRetrieved = true
	}
	// Since uuid takes priority, unlike doGetUUID, we avoid handling
	// any "no handle provided" scenarios, and return errNoUUID as well
	if !handleRetrieved {
		return errNoUUID
	}

	return nil
}

func doGetVal(c *cli.Context) error {
	if c.String("value") != "" {
		val := c.String("value")
		curr.val = &val
	}
	if curr.val == nil {
		return errNoVal
	}
	return nil
}

func doConnect(c *cli.Context) error {
	if c.String("addr") != "" {
		curr.addr = ble.NewAddr(c.String("addr"))
		curr.client = curr.clients[curr.addr.String()]
	}
	if curr.client != nil {
		return nil
	}
	return cmdConnect(c)
}

func doDiscover(c *cli.Context) error {
	if curr.profile != nil {
		return nil
	}
	return cmdDiscover(c)
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
