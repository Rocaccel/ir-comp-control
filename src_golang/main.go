package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/micmonay/keybd_event"
	"github.com/tarm/serial"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Port    string            `yaml:"port"`
	Baud    int               `yaml:"baud"`
	Buttons map[string]string `yaml:"buttons"`
}

// --- Windows API и настройки ---
var (
	// kernel32 содержит функции управления консолью и процессами
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procGetConsoleWindow = kernel32.NewProc("GetConsoleWindow") // <-- Теперь берем из kernel32
	procCreateMutex      = kernel32.NewProc("CreateMutexW")

	// user32 содержит функции графики и мыши
	user32           = syscall.NewLazyDLL("user32.dll")
	procShowWindow   = user32.NewProc("ShowWindow")
	procSetCursorPos = user32.NewProc("SetCursorPos")
	procGetCursorPos = user32.NewProc("GetCursorPos")
	procMouseEvent   = user32.NewProc("mouse_event")
)

var noDebounceActions = map[string]bool{
	"MOUSE_UP":       true,
	"MOUSE_DOWN":     true,
	"MOUSE_LEFT":     true,
	"MOUSE_RIGHT":    true,
	"VK_VOLUME_UP":   true,
	"VK_VOLUME_DOWN": true,
}

const (
	SW_HIDE              = 0
	SW_SHOW              = 5
	MOUSEEVENTF_LEFTDOWN = 0x0002
	MOUSEEVENTF_LEFTUP   = 0x0004
	ERROR_ALREADY_EXISTS = 183
)

type POINT struct{ X, Y int32 }

var vkMap = map[string]int{
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

var (
	isShuttingDown    = false
	shutdownTimer     *time.Timer
	mouseStep         int32 = 15
	lastProcessedTime time.Time
	debounceDuration  = 150 * time.Millisecond // Увеличили до 400мс для надежности
)

func moveMouse(dx, dy int32) {
	var pt POINT
	procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	procSetCursorPos.Call(uintptr(pt.X+dx), uintptr(pt.Y+dy))
}

// Проверка на запущенную копию
func checkSingleInstance() {
	mutexName, _ := syscall.UTF16PtrFromString("Local\\IRBridge_Unique_Mutex")
	ret, _, err := procCreateMutex.Call(0, 1, uintptr(unsafe.Pointer(mutexName)))
	if ret != 0 && err != nil && err.(syscall.Errno) == ERROR_ALREADY_EXISTS {
		fmt.Println("Программа уже запущена!")
		os.Exit(0)
	}
}
func main() {

	// 1. Обработка флагов
	showConsole := flag.Bool("v", false, "показывать окно консоли (режим отладки)")
	flag.Parse()

	// 2. Проверка дубликатов
	checkSingleInstance()

	// 3. Управление видимостью окна
	if !*showConsole {
		hwnd, _, _ := procGetConsoleWindow.Call()
		if hwnd != 0 {
			procShowWindow.Call(hwnd, uintptr(SW_HIDE))
		}
	}

	yamlData, err := os.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Ошибка: config.yaml не найден: %v", err)
	}

	var config Config
	if err := yaml.Unmarshal(yamlData, &config); err != nil {
		log.Fatalf("Ошибка парсинга YAML: %v", err)
	}

	sPort, err := serial.OpenPort(&serial.Config{Name: config.Port, Baud: config.Baud})
	if err != nil {
		log.Fatalf("Не удалось открыть порт %s: %v", config.Port, err)
	}
	defer sPort.Close()

	kb, _ := keybd_event.NewKeyBonding()

	fmt.Printf("Пульт готов! Слушаю порт %s...\n", config.Port)

	scanner := bufio.NewScanner(sPort)
	for scanner.Scan() {
		msg := strings.TrimSpace(scanner.Text())

		action, ok := config.Buttons[msg]
		if !ok {
			if msg != "" {
				fmt.Printf("Неизвестный код: %s\n", msg)
			}
			continue
		}
		if !noDebounceActions[action] {
			now := time.Now()
			if now.Sub(lastProcessedTime) < debounceDuration {
				continue
			}
			lastProcessedTime = now
		}
		// Логика отмены выключения
		if isShuttingDown {
			shutdownTimer.Stop()
			isShuttingDown = false
			exec.Command("powershell", "-c", "(New-Object Media.SoundPlayer 'C:\\Windows\\Media\\Windows Notify System Generic.wav').PlaySync()").Start()
			fmt.Println(">>> ВЫКЛЮЧЕНИЕ ОТМЕНЕНО")
			continue
		}

		// Логика системного выключения
		if action == "SYSTEM_SHUTDOWN" {
			isShuttingDown = true
			// Звук начала отсчета (System Exclamation)
			exec.Command("powershell", "-c", "(New-Object Media.SoundPlayer 'C:\\Windows\\Media\\Windows Exclamation.wav').PlaySync()").Start()
			fmt.Println("!!! ВЫКЛЮЧЕНИЕ ЧЕРЕЗ 10 СЕКУНД (нажмите любую кнопку для отмены) !!!")

			shutdownTimer = time.AfterFunc(10*time.Second, func() {
				fmt.Println("Завершение работы системы...")
				exec.Command("shutdown", "/s", "/t", "0").Run()
			})
			continue
		}

		// Обычная эмуляция клавиш
		if vkCode, found := vkMap[action]; found {
			fmt.Printf("Код: %s -> Клавиша: %s\n", msg, action)
			kb.SetKeys(vkCode)
			kb.Launching()
		}
		//ALT+TAB
		if action == "ALT_TAB" {
			fmt.Println("Выполняю Alt + Tab")
			kb.HasALT(true)                // Зажать Alt
			kb.SetKeys(keybd_event.VK_TAB) // Выбрать Tab
			err := kb.Launching()          // Нажать
			if err != nil {
				fmt.Println("Ошибка Alt+Tab:", err)
			}
			kb.HasALT(false) // ОТПУСТИТЬ Alt (Обязательно!)
			continue
		}
		//ALT+F4
		if action == "ALT_F4" {
			fmt.Println("Выполняю Alt + F4")
			kb.HasALT(true)               // Зажать Alt
			kb.SetKeys(keybd_event.VK_F4) // Выбрать F4
			err := kb.Launching()         // Нажать
			if err != nil {
				fmt.Println("Ошибка Alt+F4:", err)
			}
			kb.HasALT(false) // ОТПУСТИТЬ Alt (Обязательно!)
			continue
		}
		//  КОМБИНАЦИЯ WIN + D (Свернуть всё)
		if action == "WIN_D" {
			fmt.Println("Сворачиваю окна (Win+D)")
			kb.HasSuper(true)            // Зажать Win
			kb.SetKeys(keybd_event.VK_D) // Нажать D
			kb.Launching()
			kb.HasSuper(false) // Отпустить Win
			continue

		}
		//Мышь
		if action == "MOUSE_UP" {
			moveMouse(0, -mouseStep)
		}
		if action == "MOUSE_DOWN" {
			moveMouse(0, mouseStep)
		}
		if action == "MOUSE_LEFT" {
			moveMouse(-mouseStep, 0)
		}
		if action == "MOUSE_RIGHT" {
			moveMouse(mouseStep, 0)
		}
		if action == "MOUSE_CLICK" {
			procMouseEvent.Call(MOUSEEVENTF_LEFTDOWN, 0, 0, 0, 0)
			procMouseEvent.Call(MOUSEEVENTF_LEFTUP, 0, 0, 0, 0)
		}

	}
}
