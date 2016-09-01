package main

import (
	"strings"

	"github.com/currantlabs/ble"
	"github.com/urfave/cli"
)

func filter(c *cli.Context) ble.AdvFilter {
	if len(c.String("name")) != 0 {
		return func(a ble.Advertisement) bool {
			return strings.ToUpper(a.LocalName()) == strings.ToUpper(c.String("name"))
		}
	}
	if len(c.String("addr")) != 0 {
		return func(a ble.Advertisement) bool {
			return strings.ToUpper(a.Address().String()) == strings.ToUpper(c.String("addr"))
		}
	}
	return nil
}
