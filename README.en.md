# **IR comp control**

**English** | [Русский](README.md)

Lightweight Windows utility in Go that turns an IR remote into a computer controller: keyboard emulation, smart mouse with logarithmic acceleration, media commands and safe PC shutdown. Runs from the system tray with no windows.

---

## Architecture

```
IR remote → IR receiver (D2, 5V, GND, 115200) → Arduino Uno (IRremote) → COM port → Go app → WinAPI
```

The sketch `for_arduino_controller/IR-controler-dual.ino` decodes NEC/RC5 and prints lines like `NEC_36` or `RC5_36`. The app maps codes from `config.toml` to actions.

---

## Features

| Feature | Description |
| :--- | :--- |
| **Keyboard emulation** | A–Z, 0–9, F1–F12, navigation, media keys |
| **Smart mouse** | Movement with logarithmic acceleration, reset after pause >160 ms |
| **Safe shutdown** | 10-second timer cancellable by any remote button |
| **System tray** | Status, reload config, debug, quit |
| **Multi-protocol** | NEC and RC5 simultaneously |
| **Zero-GUI, single instance** | Built with `-H=windowsgui`, mutex protection |

---

## Tech stack

**App:** Go 1.26+, systray, micmonay/keybd_event, tarm/serial, BurntSushi/toml, golang.org/x/sys/windows, WinAPI user32/kernel32.
**Firmware:** Arduino Uno, IRremote 4.5.0.

---

## Requirements

**Hardware:** Arduino Uno/compatible, IR receiver TSOP4838/VS1838B, NEC/RC5 IR remote, USB cable.
**Software:** Windows 7/10/11, Arduino IDE + IRremote 4.5.0+, Go 1.26+ for building.

---

## Quick start

**Receiver wiring:** OUT → D2, VCC → 5V, GND → GND.

**Flash:**
1. Install IRremote library in Arduino IDE.
2. Open `for_arduino_controller/IR-controler-dual.ino`, select board and port, upload.
3. Open Serial Monitor at 115200, verify `NEC_XX`/`RC5_XX` lines. Close monitor before running the app.

**Run:**
1. Run `ir-comp-control_v0.1.0-win64.exe`.
2. First run creates `config.toml` next to exe — check `port`.
3. Tray menu: Reload config, Open config, Debug mode.

---

## Configuration `config.toml`

```
port = "COM4"
baud = 115200

[buttons]
"NEC_36" = "SYSTEM_SHUTDOWN"
"NEC_B"  = "VK_SPACE"
"NEC_12" = "VK_VOLUME_UP"
"NEC_3D" = "MOUSE_UP"
```

Available: `SYSTEM_SHUTDOWN`, `WIN_D`, `ALT_TAB`, `ALT_F4`, `MOUSE_UP/DOWN/LEFT/RIGHT`, `L_CLICK`, media `VK_VOLUME_*`, `VK_PLAY_PAUSE`/`NEXT`/`PREV`/`STOP`, navigation, `VK_SPACE`/`ENTER`/`ESC`/`BACKSPACE`/`TAB`/`CAPSLOCK`, digits, letters, `VK_F1`-`F12`, `VK_PRINT`, `VK_PAUSE`.

> Config is embedded via `go:embed` and extracted only if `config.toml` does not exist next to exe. Existing file is never overwritten.

---

## How features work

**Mouse acceleration:** base step 10 px. From the 4th consecutive press step grows as `10 + 60·ln(1 + (n-3))`. Reset after pause >160 ms or direction change.

**Shutdown:** system sound, 10-second countdown window. Cancel by any remote button or closing the window.

**Debounce:** 150 ms for keyboard/system commands. Mouse movement has no debounce.

---

## Debug mode

Enable «Debug mode» in tray. Button presses show message boxes with codes `NEC_XX`/`RC5_XX`. Copy code into `[buttons]`, click «Reload config».

---

## Building

```cmd
cd src_golang
go mod tidy
go build -trimpath -ldflags "-s -w -H=windowsgui" -o ..\ir-comp-control_v0.1.0-win64.exe .
```

`rsrc_windows_amd64.syso` holds the exe icon. Regenerate with `rsrc -ico icon.ico -o rsrc_windows_amd64.syso` after changing `icon.ico`.

---

## Troubleshooting

* **COM port busy** — close Arduino IDE Serial Monitor.
* **Buttons not responding** — check `NEC_`/`RC5_` prefix in `config.toml`, enable debug.
* **No data in monitor** — check D2/5V/GND, batteries, aim remote at receiver.
* **Mouse jerks** — normal at first presses, acceleration builds from 4th repeat.
* **App won't start** — another instance is running, check tray.

---

## License

BSD Zero Clause License (0BSD). [LICENSE](LICENSE).  
Firmware library: [IRremote](https://github.com/Arduino-IRremote/Arduino-IRremote).
