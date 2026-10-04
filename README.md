# Affiliate

EleBBS door that lists affiliate BBSes at log-off. Users can lightbar-select a system and telnet to it, or add their own BBS after agreeing to run this door and list you in return.

## Downloads

Prebuilt binaries are in [`releases/`](releases/):

| File | Target |
|---|---|
| [`affiliate.exe`](releases/affiliate.exe) | Windows 32-bit (EleBBS WinSock) |
| [`affiliate-linux-386`](releases/affiliate-linux-386) | Linux 32-bit |
| [`affiliate-linux-amd64`](releases/affiliate-linux-amd64) | Linux 64-bit |
| [`affiliate-linux-arm64`](releases/affiliate-linux-arm64) | Linux ARM64 |

Windows EleBBS is 32-bit. Use `affiliate.exe` built for `windows/386` so the inherited socket handle in `DOOR32.SYS` is valid.

## EleBBS setup

Run the door directly (not from a `.bat` on older Windows). Point `-T` at [TelnetDoor](https://github.com/rickparrish/TelnetDoor) and `-D` at the node dropfile.

**Windows**

```
C:\AFFILIATE\AFFILIATE.EXE -TC:\DOORS\TELNETDOOR.EXE -D*N\door32.sys
```

**Linux**

```
/bbs/affiliate/affiliate-linux-amd64 -T/bbs/doors/telnetdoor -D*N/door32.sys
```

Glued EleBBS-style switches work: `-DC:\ELEBBS\NODE1\DOOR32.SYS`

### Switches

| Switch | Meaning |
|---|---|
| `-T` / `-telnet` | Path to `telnetdoor` |
| `-D` / `-dropfile` | Path to `DOOR32.SYS`, or the node directory |
| `-C` / `-csv` | Affiliate CSV (default: `affiliates.csv` next to the binary) |
| `-ws` | Wrap the socket as gorilla/websocket frames |

## In the door

- **Up / Down** — move the lightbar
- **Enter** — run `telnetdoor -S<address> -D<dropfile> -W0`, then return to the list
- **A** — add a BBS after the affiliate agreement
- **Esc** — exit

The list is stored as CSV:

```csv
BBS Name,Telnet Address,Sysop Name,BBS Type
```

The lightbar shows 24 characters of the telnet address. The full host:port is kept in the CSV and passed to TelnetDoor.

## Building

```bat
build.bat
```

Windows 32-bit `affiliate.exe`.

```bat
build-linux.bat
```

or `./build.sh` — Linux 386, amd64, and arm64.

## License

See TelnetDoor’s license for that binary. This door is provided as-is for SysOps running EleBBS.
