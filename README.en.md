<div align="center">

<img src="icon.png" width="112" alt="IR comp control">

# IR comp control

[![Release](https://img.shields.io/github/v/release/Rocaccel/ir-comp-control?style=flat-square)](https://github.com/Rocaccel/ir-comp-control/releases/latest)
[![Go](https://img.shields.io/badge/Go-1.26+-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev)
[![Windows](https://img.shields.io/badge/platform-Windows-0078D6?style=flat-square&logo=windows&logoColor=white)](https://www.microsoft.com/windows)
[![License](https://img.shields.io/badge/license-0BSD-blue?style=flat-square)](LICENSE)

Lightweight Windows utility in Go that turns an IR remote into a computer controller: keyboard emulation, smart mouse with acceleration, media commands and safe PC shutdown. Runs from the system tray.

[Русский](README.md) | **English**

</div>

---

## Architecture

```text
IR remote → IR receiver (D2, 5V, GND, 115200) → Arduino Uno (IRremote) → COM port → Go app → WinAPI
```

The sketch `for_arduino_controller/IR-controler-dual.ino` decodes NEC/RC5 and prints lines like `NEC_36` or `RC5_36`. The app maps codes from `config.toml` to actions.

---

## Features

- **Keyboard emulation** — A–Z, 0–9, F1–F12, navigation, media keys.
- **Smart mouse** — movement with acceleration.
- **Safe shutdown** — 10-second timer, cancellable by any remote button or closing the window.
- **Live config reload** — button edits apply without restarting the app; changing the port or baud rate prompts that a restart is needed.
- **Debug mode** — one window with the button code, updated in real time.
- **Multi-protocol** — NEC and RC5 at the same time, protocol detected automatically.
- **System tray** — status, config reload and editing, quit.

---

## Requirements

- **Arduino** Uno or compatible board.
- **IR receiver** TSOP4838 / VS1838B (any 3-pin NEC receiver).
- **IR remote** NEC or RC5.
- **Windows** 7 / 10 / 11; **Go 1.26+** — only for building from source.
- **IRremote 4.x library** — for flashing the Arduino.

---

## Quick start

### 1. Wire the receiver

OUT → D2, VCC → 5V, GND → GND.

<div align="center">

![Wiring the IR receiver to Arduino: DATA → D2, VCC → 5V, GND → GND](img/ir-receiver-connection.png)

</div>

### 2. Flash the Arduino

Install the IRremote library (via Arduino IDE or `arduino-cli lib install IRremote`), open `IR-controler-dual.ino`, select the board and port, upload.

Verify in the Serial Monitor (115200): press buttons on the remote — `NEC_XX` / `RC5_XX` lines are printed.

### 3. Run

**Via Scoop** (with auto-updates):

```sh
scoop bucket add rocaccel https://github.com/Rocaccel/scoop-bucket
scoop install ir-comp-control
```

Update: `scoop update ir-comp-control`.

**Portable:** download [![Latest Release](https://img.shields.io/github/v/release/Rocaccel/ir-comp-control?label=Latest%20Release)](https://github.com/Rocaccel/ir-comp-control/releases/latest) and put it in any folder.

1. Run it — on first launch `config.toml` is created next to the exe.
2. Set your COM port in `config.toml` and run it again.

> [!IMPORTANT]
> Before launching the app, close the Arduino IDE Serial Monitor and any other programs using the COM port. Otherwise the utility won't be able to work with it.

---

## Configuration

`config.toml` next to the exe:

```toml
port = "COM4"
baud = 115200

[buttons]
"NEC_36" = "SYSTEM_SHUTDOWN"
"NEC_10" = "WIN_D"
"NEC_B"  = "ALT_TAB"
"NEC_5A" = "MOUSE_UP"
```

Button codes come from the Serial Monitor; actions from the list: `SYSTEM_SHUTDOWN`, `WIN_D`, `ALT_TAB`, `ALT_F4`, `VK_F1`–`VK_F12`, `MOUSE_UP/DOWN/LEFT/RIGHT`, `L_CLICK`, `VK_SPACE`, `VK_ENTER`, `VK_ESC`, `VK_BACKSPACE`, `VK_TAB`, `VK_CAPSLOCK`, `VK_MUTE`, `VK_VOLUME_UP/DOWN`, `VK_MEDIA_NEXT_TRACK`, `VK_MEDIA_PREV_TRACK`, `VK_MEDIA_PLAY_PAUSE`, `VK_PRINTSCREEN`, `VK_PAUSE`, `VK_HOME/END/PRIOR/NEXT/UP/DOWN/LEFT/RIGHT`, digits, letters.

> [!NOTE]
> `config.toml` is embedded in the exe and extracted next to it only on first launch — an existing file is never overwritten. Button edits are picked up by the "Reload config" button without restarting; changing `port` or `baud` requires restarting the app.

---

## How it works

**Mouse acceleration.** Base step is 10 px. From the 4th consecutive press the step grows as `10 + 60 * ln(1 + (n - 3))`. The counter resets after a pause longer than 160 ms or a direction change.

**Shutdown.** On `SYSTEM_SHUTDOWN` a sound plays and a window with a timer opens. Cancel with any remote button or by closing the window; in both cases the window closes and the PC stays on.

**Debounce.** 150 ms for all actions except mouse movement.

---

## Debug mode

Enable the "Debug mode" checkbox in the tray menu. A "Waiting for remote code" window appears: each button press shows its code (`NEC_XX` / `RC5_XX`) in real time. Copy the code into `config.toml`, save, click "Reload config". Closing the window turns the mode off.

---

## Build

```cmd
cd src_golang
go mod tidy
go build -trimpath -ldflags "-s -w -H=windowsgui -X main.version=v0.2.0" -o ..\ir-comp-control_v0.2.0-win64.exe .
```

Flags:

- `-H=windowsgui` — no console window;
- `-s -w` — smaller binary;
- `-X main.version=vX.Y.Z` — version shown in the tray menu (substitute your own).

`rsrc_windows_amd64.syso` holds the exe icon — regenerate it with `rsrc -ico icon.ico -o rsrc_windows_amd64.syso` after changing `icon.ico`.

---

## Troubleshooting

- **Button does nothing** — make sure the Serial Monitor prints `NEC_XX` / `RC5_XX` and check the codes in `config.toml`.
- **Port won't open** — close the Arduino IDE Serial Monitor and other programs using the port; check `port` and `baud` in `config.toml`.
- **Mouse jerks at first** — that's normal: acceleration builds up from the 4th press.
- **Receiver is silent** — check the D2/5V/GND wiring, remote batteries, and aim the remote at the receiver.
