# **IR comp control**

**English** | [Русский](README.md)

**IR comp control** is a lightweight Windows utility written in Go that turns a regular IR remote into a full-featured computer controller. Remote → IR receiver → Arduino → COM port → app: keyboard emulation, smart mouse with acceleration, media commands, and safe PC shutdown — all from the system tray, without extra windows.

---

## Table of Contents

1. [Overview](#overview)
2. [How It Works](#how-it-works)
3. [Features](#features)
4. [Tech Stack](#tech-stack)
5. [Requirements](#requirements)
6. [Quick Start](#quick-start)
7. [Configuration (config.toml)](#configuration-configtoml)
8. [Important Notes on Config & Migration](#important-notes-on-config--migration)
9. [How Features Work](#how-features-work)
10. [Debug Mode](#debug-mode)
11. [Building from Source](#building-from-source)
12. [Repository Structure](#repository-structure)
13. [Troubleshooting](#troubleshooting)
14. [License](#license)

---

## Overview

The project consists of two parts:

- **Arduino firmware** (`for_arduino_controller/IR-controler-dual.ino`) — receives signals from an IR receiver and forwards button codes to the serial port. Supports **NEC** and **RC5** protocols simultaneously with automatic detection.
- **Windows application** (`src_golang/`) — reads codes from the COM port and performs actions: keyboard emulation, mouse movement, system commands.

Both parts communicate over a simple text protocol: each line is a button code in the format `NEC_XX` or `RC5_XX` (hexadecimal command number).

---

## How It Works

```
┌──────────┐  IR signal   ┌──────────────┐   Serial    ┌─────────────────┐   WinAPI   ┌──────────┐
│ IR remote│ ───────────→ │ IR receiver  │ ──────────→ │ Arduino Uno     │ ─────────→ │ Go app   │ ──→ Keyboard / Mouse / System
│          │              │ (D2, 5V, GND)│  115200     │ (dual sketch)   │  COM port  │          │
└──────────┘              └──────────────┘             └─────────────────┘            └──────────┘
```

1. The **remote** sends an IR signal (NEC or RC5 protocol).
2. The **IR receiver** (e.g., TSOP4838, VS1838B) demodulates the signal and outputs it to Arduino pin **D2**.
3. The **sketch** `IR-controler-dual.ino` (IRremote library) decodes the signal, auto-detects the protocol, and prints a line like `NEC_36` or `RC5_36` to Serial.
4. The **Go application** reads lines from the COM port, looks up the code in `config.toml`, and executes the associated action via WinAPI.

---

## Features

| Feature | Description |
| :---- | :---- |
| **Keyboard emulation** | Letters A–Z, digits 0–9, F1–F12, navigation (arrows, Home/End, PgUp/PgDn, Insert/Delete), media keys (volume, play/pause, next/previous track, stop) |
| **Smart mouse** | Cursor movement with logarithmic acceleration — the longer you hold, the faster it goes |
| **Safe shutdown** | 10-second timer with cancellation by any remote button or closing the window |
| **System tray** | Tray icon with menu: status, reload config, open config, debug mode, quit |
| **Protocol auto-detection** | NEC and RC5 work simultaneously, no switching needed |
| **Debug mode** | Shows raw remote codes for button binding |
| **Zero-GUI** | No console window at runtime (built with `-H=windowsgui`) |
| **Single instance** | Mutex prevents running the app twice |

---

## Tech Stack

**Application (Windows):**

- **Language:** Go 1.26+
- **systray** — system tray icon and menu
- **micmonay/keybd_event** — keyboard emulation
- **tarm/serial** — COM port reading
- **BurntSushi/toml** — configuration parsing
- **golang.org/x/sys/windows** — typed WinAPI wrappers
- **WinAPI (user32/kernel32)** — cursor, mouse, messages, mutex

**Firmware (Arduino):**

- **Arduino Uno** (and ATmega328P-compatible boards)
- **IRremote 4.5.0** — IR protocol decoding (NEC, RC5, and others)

---

## Requirements

**Hardware:**

- Arduino Uno or compatible board (ATmega328P)
- Three-pin IR receiver (TSOP4838, VS1838B, and similar)
- IR remote with NEC or RC5 protocol (almost any household remote works)
- USB cable to connect Arduino to the computer

**Software:**

- Windows (7/10/11)
- Arduino IDE (for flashing the sketch)
- Go 1.26+ (only if building from source)

---

## Quick Start

### 1. Wiring the IR Receiver to Arduino

| Receiver pin | Connect to |
| :---- | :---- |
| OUT (signal) | D2 |
| VCC (power) | 5V |
| GND (ground) | GND |

The sketch expects the signal on pin **D2**. Check your specific receiver's pinout before connecting — pin order varies between models. USB power is sufficient; no external supply needed.

### 2. Installing the IRremote Library

1. Open Arduino IDE.
2. Go to *Tools → Manage Libraries…* (or `Ctrl+Shift+I`).
3. Search for **IRremote** and install version **4.5.0** (or newer).

### 3. Flashing the Sketch

1. Open `for_arduino_controller/IR-controler-dual.ino` in Arduino IDE.
2. Select the board: *Tools → Board → Arduino Uno*.
3. Select the port: *Tools → Port → COMx* (your Arduino).
4. Click *Upload* (right arrow).

### 4. Verifying Reception

1. Open the *Serial Monitor* (magnifying glass, top right) at **115200** baud.
2. Point the remote at the receiver and press buttons.
3. Lines like `NEC_36` or `RC5_36` should appear — the prefix depends on the remote's protocol.

> On startup, the sketch prints `Arduino IR to Serial Bridge Ready (NEC + RC5 auto-detect)` — the app silently ignores this line, which is normal.

### 5. Running the Application

1. Run `ir-comp-control_v0.1.0-win64.exe`.
2. On first launch, `config.toml` is created next to the exe — open it and check the `port` parameter (must match the Arduino's port, e.g. `"COM4"`).
3. The app minimizes to the tray. Press remote buttons — actions are executed.

> **Important:** close the Arduino IDE Serial Monitor before running the app — otherwise the port is busy.

---

## Configuration (config.toml)

The configuration file is created automatically on first launch. Format: TOML.

### Main Parameters

| Parameter | Description | Example |
| :---- | :---- | :---- |
| `port` | COM port name where Arduino is connected | `"COM4"` |
| `baud` | Baud rate (must match the sketch) | `115200` |

### `[buttons]` Section

Maps remote button codes to system actions. Codes are prefixed with the protocol (`NEC_` or `RC5_`), which is auto-detected.

```toml
[buttons]
"NEC_36" = "SYSTEM_SHUTDOWN"   # Power button → PC shutdown
"NEC_B"  = "VK_SPACE"         # Space emulation
"NEC_12" = "VK_VOLUME_UP"     # Volume +
"NEC_13" = "VK_VOLUME_DOWN"   # Volume −
"NEC_3D" = "MOUSE_UP"         # Mouse movement up
```

### Available Commands

| Category | Commands |
| :---- | :---- |
| **System** | `SYSTEM_SHUTDOWN`, `WIN_D`, `ALT_TAB`, `ALT_F4` |
| **Mouse** | `MOUSE_UP`, `MOUSE_DOWN`, `MOUSE_LEFT`, `MOUSE_RIGHT`, `L_CLICK` |
| **Media** | `VK_VOLUME_UP`, `VK_VOLUME_DOWN`, `VK_VOLUME_MUTE`, `VK_PLAY_PAUSE`, `VK_NEXT`, `VK_PREV`, `VK_STOP` |
| **Navigation** | `VK_LEFT`, `VK_RIGHT`, `VK_UP`, `VK_DOWN`, `VK_PAGEUP`, `VK_PAGEDOWN`, `VK_HOME`, `VK_END`, `VK_INSERT`, `VK_DELETE` |
| **Basic** | `VK_SPACE`, `VK_ENTER`, `VK_ESC`, `VK_BACKSPACE`, `VK_TAB`, `VK_CAPSLOCK` |
| **Digits** | `VK_0` … `VK_9` |
| **Letters** | `VK_A` … `VK_Z` |
| **Function keys** | `VK_F1` … `VK_F12` |
| **Other** | `VK_PRINT`, `VK_PAUSE` |

---

## Important Notes on Config & Migration

The configuration file is **embedded in the exe** via `go:embed` and extracted on first launch — **only if there is no `config.toml` next to the application**.

> ⚠️ An existing `config.toml` is **never overwritten**. If you updated the exe but an old config file is present, the app will keep using the old file.

**Migration from older versions:** after flashing the dual sketch, button codes changed from `KEY_xx` to `NEC_xx` / `RC5_xx`. If you have an old config:

- **Option 1:** delete `config.toml` — a new one will be created from the embedded template on next launch.
- **Option 2:** manually replace the `KEY_` prefix with `NEC_` (or `RC5_`) in all `[buttons]` entries.

---

## How Features Work

### Mouse Acceleration

The app uses an adaptive cursor movement algorithm:

- **Base step:** 10 pixels per press.
- **Acceleration:** starting from the 4th consecutive press, the step grows as `10 + 60 · ln(1 + (n − 3))`, where `n` is the accumulated signal count.
- **Reset:** the counter resets after a pause longer than 160 ms or on direction change.

### Shutdown Algorithm

1. On receiving `SYSTEM_SHUTDOWN`, a system sound plays.
2. A window with a 10-second countdown appears.
3. **To cancel:**
   - Press **any** button on the remote.
   - Or close the window on screen.
4. On cancel, a soft beep sounds and the timer stops.

### Debounce

Keyboard and system commands have a 150 ms debounce — accidental repeats while holding a button are filtered out. Mouse movement and clicks run without debounce for smooth motion.

---

## Debug Mode

If remote buttons don't respond:

1. Enable **"Debug Mode"** in the tray menu (checkbox).
2. Press buttons on the remote — the app shows windows with received codes (e.g., `Received: NEC_A1` or `Received: RC5_A1`).
3. Copy the code and add it to the `[buttons]` section of `config.toml`.
4. Click **"Reload Config"** in the tray menu — changes apply without restart.

---

## Building from Source

Requirements: Windows, Go 1.26+.

All dependencies are fetched automatically via `go mod tidy`.

```cmd
cd src_golang
go mod tidy
go build -trimpath -ldflags "-s -w -H=windowsgui" -o ..\ir-comp-control_v0.1.0-win64.exe .
```

Build flags:

| Flag | Purpose |
| :---- | :---- |
| `-trimpath` | Removes absolute paths from the binary |
| `-s -w` | Strips debug info and symbol table (smaller size) |
| `-H=windowsgui` | Builds without a console window |

> **About the exe icon:** `rsrc_windows_amd64.syso` contains the icon resources for Explorer — **do not delete it**, otherwise the built exe will have no icon. If you changed `icon.ico`, regenerate the resources:
>
> ```cmd
> rsrc -ico icon.ico -o rsrc_windows_amd64.syso
> ```

---

## Repository Structure

```
IR_comp_control/
├── for_arduino_controller/
│   └── IR-controler-dual.ino      # Arduino sketch (NEC + RC5, auto-detect)
├── src_golang/
│   ├── main.go                    # All application code
│   ├── go.mod / go.sum            # Module and dependencies
│   ├── icon.ico                   # Tray icon (embedded via go:embed)
│   ├── for-icons.png              # Icon source
│   ├── rsrc_windows_amd64.syso    # Exe icon resources for Explorer
│   └── config.toml                # Config template (embedded via go:embed)
├── LICENSE                        # BSD Zero Clause License
├── README.md                      # Documentation (Russian)
└── README.en.md                   # Documentation (English)
```

---

## Troubleshooting

| Problem | Solution |
| :---- | :---- |
| **"Failed to open COM port"** | Close the Arduino IDE Serial Monitor — the port is busy. Check that the board is connected and the driver is installed. |
| **Buttons don't respond** | Enable debug mode and check which codes arrive. Make sure the prefix in `config.toml` (`NEC_`/`RC5_`) matches reality. |
| **Silence in Serial Monitor** | Check receiver wiring (D2/5V/GND), remote batteries, and point the remote directly at the receiver. |
| **Mouse jerks** | Normal at first presses — acceleration grows from the 4th repeat. Speed resets after a 160 ms pause. |
| **Shutdown won't cancel** | Cancellation works with any remote button or closing the window. Verify the sketch sends codes (check with Serial Monitor). |
| **App won't start** | Another instance may already be running (mutex). Check the tray and Task Manager. |

---

## License

BSD Zero Clause License (0BSD). See the [LICENSE](LICENSE) file for details.

The microcontroller firmware uses the [IRremote](https://github.com/Arduino-IRremote/Arduino-IRremote) library.
