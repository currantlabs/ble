A go clone of python-tikteck (https://github.com/mjg59/python-tikteck)
(not working yet)

```
  sudo go run *.go
```

```
Service: 000102030405060708090A0B0C0D1910 , Handle (0x10)
  Characteristic: 000102030405060708090A0B0C0D1911, Property: 0x1A (RWN), , Handle(0x11), VHandle(0x12)
    Value         01 | "\x01"
    Descriptor: 2901, Characteristic User Description, Handle(0x13)
    Value         537461747573 | "Status"

  Characteristic: 000102030405060708090A0B0C0D1912, Property: 0x0E (wWR), , Handle(0x14), VHandle(0x15)
    Value         00000000000000000000000000000000 | "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"
    Descriptor: 2901, Characteristic User Description, Handle(0x16)
    Value         436f6d6d616e64 | "Command"

  Characteristic: 000102030405060708090A0B0C0D1913, Property: 0x06 (Rw), , Handle(0x17), VHandle(0x18)
    Value         e0000000000000000000000000000000 | "\xe0\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"
    Descriptor: 2901, Characteristic User Description, Handle(0x19)
    Value         4f5441 | "OTA"

  Characteristic: 000102030405060708090A0B0C0D1914, Property: 0x0A (RW), , Handle(0x1A), VHandle(0x1B)
    Value         007857b58d2af70eb269e149192262b66b | "\x00xW\xb5\x8d*\xf7\x0e\xb2i\xe1I\x19\"b\xb6k"
    Descriptor: 2901, Characteristic User Description, Handle(0x1c)
    Value         50616972 | "Pair"

Service: 19200D0C0B0A09080706050403020100 , Handle (0x1D)
  Characteristic: 19210D0C0B0A09080706050403020100, Property: 0x0A (RW), , Handle(0x1E), VHandle(0x1F)
    Value         d0000000000000000000000000000000 | "\xd0\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"
    Descriptor: 2901, Characteristic User Description, Handle(0x20)
    Value         534c434d44 | "SLCMD"
```
