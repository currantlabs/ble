
``` blesh -h ```

```
NAME:
   blesh - A CLI tool for ble

USAGE:
   blesh [global options] command [command options] [arguments...]

VERSION:
   0.0.1

COMMANDS:
     scan, s    Scan surrounding with specified filter
     exp, e     Scan and explore surrounding with specified filter
     adv, a     Advertise name, UUIDs, iBeacon (TODO)
     serve, sv  Start the GATT Server
     shell, sh  Entering interactive mode
     help, h    Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --device value  implementation of ble (default / bled) (default: "default")
   --help, -h      show help
   --version, -v   print the version
```

```
# Scan for device that has name of "Gopher" for 10 seconds.
blesh scan -n Gopher -d 10s

# Advertise name of Gopher for 10 seconds.
blesh adv -n Gopher -d 10s

# Advertise iBeacon.
(TODO)

# Advertise crafted advertising data packet (Linux only).
(TODO)

# Explore a device with specified mac address (or Device UUID on OS X)
blesh exp -addr ADDR

# Start a GATT server for 10m.
blesh serve -d 10m


# (TODO) Client functions, such as connect, read/write/subscribe/...

```
