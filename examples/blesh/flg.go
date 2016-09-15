package main

import (
	"time"

	"github.com/urfave/cli"
)

var (
	flgTimeout  = cli.DurationFlag{Name: "tmo, t", Value: time.Second * 5, Usage: "Timeout for the command"}
	flgName     = cli.StringFlag{Name: "name, n", Usage: "Name of remote device"}
	flgAddr     = cli.StringFlag{Name: "addr, a", Usage: "Address of remote device"}
	flgSvc      = cli.StringFlag{Name: "svc, s", Usage: "Services of remote device"}
	flgAllowDup = cli.BoolFlag{Name: "dup", Usage: "Allow duplicate in scanning result"}
	flgUUID     = cli.StringFlag{Name: "uuid, u", Usage: "UUID"}
	flgInd      = cli.BoolFlag{Name: "ind", Usage: "Indication"}
)
