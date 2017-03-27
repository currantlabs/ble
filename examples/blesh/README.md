
``` blesh -h ```

```
NAME:
   blesh - A CLI tool for ble

USAGE:
   blesh [global options] command [command options] [arguments...]

VERSION:
   0.0.1

COMMANDS:
     status, st     Display current status
     adv, a         Advertise name, UUIDs, iBeacon (TODO)
     serve, sv      Start the GATT Server
     scan, s        Scan surrounding with specified filter
     connect, c     Connect to a peripheral device
     disconnect, x  Disconnect a connected peripheral device
     discover, d    Discover profile on connected device
     explore, e     Display discovered profile
     read, r        Read value from a characteristic or descriptor
     write, w       Write value to a characteristic or descriptor
     sub            Subscribe to notification (or indication)
     unsub          Unsubscribe to notification (or indication)
     shell, sh      Enter interactive mode
     help, h        Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --help, -h      show help
   --version, -v   print the version
```

*** Quick Tutorial ***

We will start two instances of blesh; one as a peripheral running a simple
GATT server while another one acting as client connecting to the server, and explore it's profile.

** Build **

blesh supports both OS X and Linux, and can be cross-compiled easily.
Let's build two binaries and have them running on different platforms.
(Or you can run two instances on a Linux if you have more than one ble devices.)
```
GOOS=darwin go build -o blesh_osx *.go
GOOS=linux go build -o blesh_lnx *.go
```

Start a GATT server on the Linux platform (with a usb ble dongle) for 1 hour.
(We'll leave it sit there for the rest of tutoruial)
```
sudo ./blesh_lnx sv -tmo 1h

Initializing device ...
Serving GATT Server for 1h0m0s...
```

Start another instance on the OS X, or Linux with another ble device. Use the shell subcommand to enter interactive mode.

```
./blesh_osx shell

Initializing device ...
blesh >
```

Scan for surrounding devices

```
blesh > scan -tmo 2s
Scanning for 2s...
[41a2e12c9265407a82e25379df839e0d] C -47: Name: Gopher, Svcs: [0001000000011000800000805F9B34FB]
[41a2e12c9265407a82e25379df839e0d] C -47: Name: Gopher
[41a2e12c9265407a82e25379df839e0d] C -47: Name: Gopher, Svcs: [0001000000011000800000805F9B34FB]
[41a2e12c9265407a82e25379df839e0d] C -48: Name: Gopher
[1a7026ac7be8402680926d95ef2f0661] C  86: Name: Bose QuietComfort 35, Svcs: [FEBE], MD: 1001400C0141ACBC32794E58
[1a7026ac7be8402680926d95ef2f0661] C  86: Name: Bose QuietComfort 35
[41a2e12c9265407a82e25379df839e0d] C -48: Name: Gopher, Svcs: [0001000000011000800000805F9B34FB]
[41a2e12c9265407a82e25379df839e0d] C -48: Name: Gopher
...
```

The peripheral instance is advertising with "Gopher" as its name, and a service with UUID "0001000000011000800000805F9B34FB".

Now, let's scan only for device that has name of "Gopher" for 1 second. And don't show duplicate.

```
blesh > scan -name Gopher -dup=false -tmo 1s
Scanning for 1s...
[41a2e12c9265407a82e25379df839e0d] C -48: Name: Gopher, Svcs: [0001000000011000800000805F9B34FB]
[41a2e12c9265407a82e25379df839e0d] C -58: Name: Gopher
```

The result is much cleaner now.

```
blesh > status
Current status:
  Initialized: yes
  Address:     41a2e12c9265407a82e25379df839e0d
  Connected:
  Profile:
  UUID:
```

Status shows some cached values, which can serve as default values in some commands. For example, now we can issue "connect" command without explicitly specifying which device to connect.

```
blesh > connect
Connecting ...
```

Or we can still connect to device with specified condition.

```
blesh > disconnect
Disconnecting [ 41A2E12C9265407A82E25379DF839E0D ]... (this might take up to few seconds on OS X)

blesh > connect -addr 41A2E12C9265407A82E25379DF839E0D
Connecting ...
```

```
blesh > status
Current status:
  Initialized: yes
  Address:     41A2E12C9265407A82E25379DF839E0D
  Connected:   41A2E12C9265407A82E25379DF839E0D
  Profile:
  UUID:
```

Now we're connected, and let's discover the profile (service/char/...) on the server.

```
blesh > discover
Discovering profile...
```

```
blesh > status
Current status:
  Initialized: yes
  Address:     41A2E12C9265407A82E25379DF839E0D
  Connected:   41A2E12C9265407A82E25379DF839E0D
  Profile:

    Service: 0001000000011000800000805F9B34FB , Handle (0x10)
      Characteristic: 0001000000021000800000805F9B34FB , Property: 0x3E (wWNIR), Handle(0x11), VHandle(0x12)
        Value         636f756e743a20526561642030 | "count: Read 0"
        Descriptor: 2902 Client Characteristic Configuration, Handle(0x13)
        Value         0000 | "\x00\x00"
      Characteristic: 0002000000021000800000805F9B34FB , Property: 0x3C (IwWN), Handle(0x14), VHandle(0x15)
        Descriptor: 2902 Client Characteristic Configuration, Handle(0x16)
        Value         0000 | "\x00\x00"

  UUID:
```

To display the profile only, use the explore command.
```
blesh > explore
Discovering profile...
    Service: 0001000000011000800000805F9B34FB , Handle (0x10)
      Characteristic: 0001000000021000800000805F9B34FB , Property: 0x3E (IRwWN), Handle(0x11), VHandle(0x12)
        Value         636f756e743a20526561642031 | "count: Read 1"
        Descriptor: 2902 Client Characteristic Configuration, Handle(0x13)
        Value         0000 | "\x00\x00"
      Characteristic: 0002000000021000800000805F9B34FB , Property: 0x3C (wWNI), Handle(0x14), VHandle(0x15)
        Descriptor: 2902 Client Characteristic Configuration, Handle(0x16)
        Value         0000 | "\x00\x00"
```

Notice that the value read from characteristic is changed (our sample characteristic simply increment the counter when it is read).

Use the "read" command to read the characteristic value again.

```
blesh > read -uuid 0001000000021000800000805F9B34FB
    Value         636f756e743a20526561642032 | "count: Read 2"
```

```
blesh > st
Current status:
  Initialized: yes
  Address:     41A2E12C9265407A82E25379DF839E0D
  Connected:   41A2E12C9265407A82E25379DF839E0D
  Profile:

    Service: 0001000000011000800000805F9B34FB , Handle (0x10)
      Characteristic: 0001000000021000800000805F9B34FB , Property: 0x3E (RwWNI), Handle(0x11), VHandle(0x12)
        Value         636f756e743a20526561642033 | "count: Read 3"
        Descriptor: 2902 Client Characteristic Configuration, Handle(0x13)
        Value         0000 | "\x00\x00"
      Characteristic: 0002000000021000800000805F9B34FB , Property: 0x3C (wWNI), Handle(0x14), VHandle(0x15)
        Descriptor: 2902 Client Characteristic Configuration, Handle(0x16)
        Value         0000 | "\x00\x00"

  UUID:       0001000000021000800000805F9B34FB
```

Notice now the UUID has been cached, too. So, we can "subscribe" (or unsubscribe) to the characteristic without explicitly specifying it.

```
blesh > sub
blesh >
notified: 436f756e743a2030 | "Count: 0"
notified: 436f756e743a2031 | "Count: 1"
notified: 436f756e743a2032 | "Count: 2"
notified: 436f756e743a2033 | "Count: 3"
notified: 436f756e743a2034 | "Count: 4"
unsub
blesh >
```

Disconnect, and quit the shell.

```
blesh > disconnect
Disconnecting [ 41A2E12C9265407A82E25379DF839E0D ]... (this might take up to few seconds on OS X)

blesh > quit
```

Given that we've know what filter to use for connecting, and which UUID to manipulate, we can use the one-liner to read the value directly.

```
./blesh_osx read -name Gopher -uuid 0001000000021000800000805F9B34FB
Initializing device ...
Connecting...
Discovering profile...
    Value         636f756e743a20526561642034 | "count: Read 4"
```
