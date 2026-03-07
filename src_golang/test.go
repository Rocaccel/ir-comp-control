package main

import (
	"bufio"
	_ "embed"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/getlantern/systray"
	"github.com/micmonay/keybd_event"
	"github.com/tarm/serial"
	"gopkg.in/yaml.v3"
)

// --- Встраивание ресурсов ---
//
//go:embed icon.ico
var iconBytes []byte

//go:embed config.yaml
var defaultYaml []byte

// --- Windows API ---
var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	user32           = syscall.NewLazyDLL("user32.dll")
	procCreateMutex  = kernel32.NewProc("CreateMutexW")
	procMessageBox   = user32.NewProc("MessageBoxW")
	procSetCursorPos = user32.NewProc("SetCursorPos")
	procGetCursorPos = user32.NewProc("GetCursorPos")
	procMouseEvent   = user32.NewProc("mouse_event")
)

const (
	ERROR_ALREADY_EXISTS = 183
	MOUSEEVENTF_LEFTDOWN = 0x0002
	MOUSEEVENTF_LEFTUP   = 0x0004
	MB_OK                = 0x00000000
	MB_ICONINFORMATION   = 0x00000040
	MB_TOPMOST           = 0x00040000
)

type POINT struct{ X, Y int32 }

// --- Глобальные переменные ---
var (
	isShuttingDown    = false
	shutdownTimer     *time.Timer
	mouseStep         int32 = 15
	lastProcessedTime time.Time
	debounceDuration  = 150 * time.Millisecond
	debugMode         = false
	reloadChan        = make(chan bool)
)

func main() {
	checkSingleInstance()

	// Самораспаковка конфига через современный пакет os
	if _, err := os.Stat("config.yaml"); os.IsNotExist(err) {
		_ = os.WriteFile("config.yaml", defaultYaml, 0644)
	}

	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetIcon(iconBytes)
	systray.SetTitle("IR Remote Control")
	systray.SetTooltip("Управление пультом активно")

	systray.AddMenuItem("Статус: Работает", "").Disable()

	mReload := systray.AddMenuItem("Перезагрузить конфиг", "Применить изменения")
	mOpenConfig := systray.AddMenuItem("Открыть конфиг", "Редактировать кнопки")
	mDebug := systray.AddMenuItemCheckbox("Режим отладки", "Показывать коды", false)
	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Выход", "")

	go func() {
		for {
			select {
			case <-mReload.ClickedCh:
				exec.Command("powershell", "-c", "[System.Media.SystemSounds]::Asterisk.Play()").Run()
				reloadChan <- true
			case <-mDebug.ClickedCh:
				if mDebug.Checked() {
					mDebug.Uncheck()
					debugMode = false
				} else {
					mDebug.Check()
					debugMode = true
				}
			case <-mOpenConfig.ClickedCh:
				exec.Command("notepad.exe", "config.yaml").Start()
			case <-mQuit.ClickedCh:
				systray.Quit()
			}
		}
	}()

	go worker()
}

func worker() {
	for {
		stopThisSession := make(chan bool)
		go runRemoteControl(stopThisSession)

		<-reloadChan
		stopThisSession <- true
		time.Sleep(500 * time.Millisecond)
	}
}

func runRemoteControl(stopSignal chan bool) {
	yamlData, err := os.ReadFile("config.yaml")
	if err != nil {
		return
	}
	var config struct {
		Port    string            `yaml:"port"`
		Baud    int               `yaml:"baud"`
		Buttons map[string]string `yaml:"buttons"`
	}
	_ = yaml.Unmarshal(yamlData, &config)

	sPort, err := serial.OpenPort(&serial.Config{Name: config.Port, Baud: config.Baud})
	if err != nil {
		showDebugBox("Ошибка порта", "Не удалось открыть "+config.Port)
		return
	}
	defer sPort.Close()

	kb, _ := keybd_event.NewKeyBonding()
	scanner := bufio.NewScanner(sPort)

	go func() {
		<-stopSignal
		sPort.Close()
	}()

	for scanner.Scan() {
		msg := strings.TrimSpace(scanner.Text())
		if msg == "" {
			continue
		}
		if debugMode {
			showDebugBox("ИК Код", "Получен: "+msg)
		}

		action, ok := config.Buttons[msg]
		if !ok {
			continue
		}

		if !strings.HasPrefix(action, "MOUSE_") {
			if time.Since(lastProcessedTime) < debounceDuration {
				continue
			}
			lastProcessedTime = time.Now()
		}

		handleAction(action, kb)
	}
}

func handleAction(action string, kb keybd_event.KeyBonding) {
	// HEX-коды клавиш (надежнее, чем константы библиотеки)
	vkMap := map[string]int{
		// --- УПРАВЛЕНИЕ СИСТЕМОЙ ---
		"VK_SPACE":     keybd_event.VK_SPACE,
		"VK_ENTER":     keybd_event.VK_ENTER,
		"VK_ESC":       keybd_event.VK_ESC,
		"VK_BACKSPACE": keybd_event.VK_BACKSPACE,
		"VK_TAB":       keybd_event.VK_TAB,
		"VK_CAPSLOCK":  keybd_event.VK_CAPSLOCK,

		// --- НАВИГАЦИЯ ---
		"VK_LEFT":     keybd_event.VK_LEFT,
		"VK_RIGHT":    keybd_event.VK_RIGHT,
		"VK_UP":       keybd_event.VK_UP,
		"VK_DOWN":     keybd_event.VK_DOWN,
		"VK_PAGEUP":   keybd_event.VK_PAGEUP,
		"VK_PAGEDOWN": keybd_event.VK_PAGEDOWN,
		"VK_HOME":     keybd_event.VK_HOME,
		"VK_END":      keybd_event.VK_END,
		"VK_INSERT":   keybd_event.VK_INSERT,
		"VK_DELETE":   keybd_event.VK_DELETE,

		// --- МУЛЬТИМЕДИА ---
		"VK_VOLUME_UP":   keybd_event.VK_VOLUME_UP,
		"VK_VOLUME_DOWN": keybd_event.VK_VOLUME_DOWN,
		"VK_VOLUME_MUTE": keybd_event.VK_VOLUME_MUTE,
		"VK_PLAY_PAUSE":  keybd_event.VK_MEDIA_PLAY_PAUSE,
		"VK_NEXT":        keybd_event.VK_MEDIA_NEXT_TRACK,
		"VK_PREV":        keybd_event.VK_MEDIA_PREV_TRACK,
		"VK_STOP":        keybd_event.VK_MEDIA_STOP,

		// --- ЦИФРЫ (0-9) ---
		"VK_0": keybd_event.VK_0, "VK_1": keybd_event.VK_1, "VK_2": keybd_event.VK_2,
		"VK_3": keybd_event.VK_3, "VK_4": keybd_event.VK_4, "VK_5": keybd_event.VK_5,
		"VK_6": keybd_event.VK_6, "VK_7": keybd_event.VK_7, "VK_8": keybd_event.VK_8,
		"VK_9": keybd_event.VK_9,

		// --- БУКВЫ (A-Z) ---
		"VK_A": keybd_event.VK_A, "VK_B": keybd_event.VK_B, "VK_C": keybd_event.VK_C,
		"VK_D": keybd_event.VK_D, "VK_E": keybd_event.VK_E, "VK_F": keybd_event.VK_F,
		"VK_G": keybd_event.VK_G, "VK_H": keybd_event.VK_H, "VK_I": keybd_event.VK_I,
		"VK_J": keybd_event.VK_J, "VK_K": keybd_event.VK_K, "VK_L": keybd_event.VK_L,
		"VK_M": keybd_event.VK_M, "VK_N": keybd_event.VK_N, "VK_O": keybd_event.VK_O,
		"VK_P": keybd_event.VK_P, "VK_Q": keybd_event.VK_Q, "VK_R": keybd_event.VK_R,
		"VK_S": keybd_event.VK_S, "VK_T": keybd_event.VK_T, "VK_U": keybd_event.VK_U,
		"VK_V": keybd_event.VK_V, "VK_W": keybd_event.VK_W, "VK_X": keybd_event.VK_X,
		"VK_Y": keybd_event.VK_Y, "VK_Z": keybd_event.VK_Z,

		// --- ФУНКЦИОНАЛЬНЫЕ (F1-F12) ---
		"VK_F1": keybd_event.VK_F1, "VK_F2": keybd_event.VK_F2, "VK_F3": keybd_event.VK_F3,
		"VK_F4": keybd_event.VK_F4, "VK_F5": keybd_event.VK_F5, "VK_F6": keybd_event.VK_F6,
		"VK_F7": keybd_event.VK_F7, "VK_F8": keybd_event.VK_F8, "VK_F9": keybd_event.VK_F9,
		"VK_F10": keybd_event.VK_F10, "VK_F11": keybd_event.VK_F11, "VK_F12": keybd_event.VK_F12,

		// --- ПРОЧЕЕ ---
		"VK_PRINT": keybd_event.VK_PRINT,
		"VK_PAUSE": keybd_event.VK_PAUSE,
	}

	if isShuttingDown {
		shutdownTimer.Stop()
		isShuttingDown = false
		exec.Command("powershell", "-c", "(New-Object Media.SoundPlayer 'C:\\Windows\\Media\\Windows Background.wav').Play()").Start()
		return
	}

	switch action {
	case "SYSTEM_SHUTDOWN":
		isShuttingDown = true
		exec.Command("powershell", "-c", "(New-Object Media.SoundPlayer 'C:\\Windows\\Media\\chimes.wav').Play()").Start()
		shutdownTimer = time.AfterFunc(10*time.Second, func() {
			exec.Command("shutdown", "/s", "/t", "0").Run()
		})
	case "MOUSE_UP":
		moveMouse(0, -mouseStep)
	case "MOUSE_DOWN":
		moveMouse(0, mouseStep)
	case "MOUSE_LEFT":
		moveMouse(-mouseStep, 0)
	case "MOUSE_RIGHT":
		moveMouse(mouseStep, 0)
	case "L_CLICK":
		procMouseEvent.Call(MOUSEEVENTF_LEFTDOWN, 0, 0, 0, 0)
		procMouseEvent.Call(MOUSEEVENTF_LEFTUP, 0, 0, 0, 0)
	case "WIN_D":
		kb.HasSuper(true)
		kb.SetKeys(keybd_event.VK_D)
		kb.Launching()
		kb.HasSuper(false)
	case "ALT_TAB":
		kb.HasALT(true)
		kb.SetKeys(keybd_event.VK_TAB)
		kb.Launching()
		kb.HasALT(false)
	default:
		if vkCode, found := vkMap[action]; found {
			kb.SetKeys(vkCode)
			kb.Launching()
		}
	}
}

func showDebugBox(title, text string) {
	t, _ := syscall.UTF16PtrFromString(text)
	c, _ := syscall.UTF16PtrFromString(title)
	go procMessageBox.Call(0, uintptr(unsafe.Pointer(t)), uintptr(unsafe.Pointer(c)), MB_OK|MB_ICONINFORMATION|MB_TOPMOST)
}

func moveMouse(dx, dy int32) {
	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procSetCursorPos.Call(uintptr(pt.X+dx), uintptr(pt.Y+dy))
}

func checkSingleInstance() {
	mutexName, _ := syscall.UTF16PtrFromString("Local\\IRRemote")
	ret, _, err := procCreateMutex.Call(0, 1, uintptr(unsafe.Pointer(mutexName)))
	if ret != 0 && err != nil && err.(syscall.Errno) == ERROR_ALREADY_EXISTS {
		os.Exit(0)
	}
}

func onExit() { os.Exit(0) }
