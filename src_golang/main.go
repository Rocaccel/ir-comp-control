package main

import (
	"bufio"
	_ "embed"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/BurntSushi/toml"
	"github.com/getlantern/systray"
	"github.com/micmonay/keybd_event"
	"github.com/tarm/serial"
	"golang.org/x/sys/windows"
)

// --- Встраивание ресурсов ---

//go:embed icon.ico
var iconBytes []byte

//go:embed config.toml
var defaultToml []byte

// --- Windows API ---
// Большая часть WinAPI вызывается через типизированные обёртки
// golang.org/x/sys/windows. Через raw-вызовы осталось только то,
// чего в пакете windows нет: курсор, mouse_event и MessageBeep.
var (
	user32           = windows.NewLazySystemDLL("user32.dll")
	procSetCursorPos = user32.NewProc("SetCursorPos")
	procGetCursorPos = user32.NewProc("GetCursorPos")
	procMouseEvent   = user32.NewProc("mouse_event")
	procMessageBeep  = user32.NewProc("MessageBeep")
)

const (
	mouseLeftDown = 0x0002
	mouseLeftUp   = 0x0004

	// Звук MessageBeep без иконки — обычный системный "писк".
	beepSimple = 0xFFFFFFFF

	shutdownPrivilege = "SeShutdownPrivilege"
	shutdownTimeout   = 10 * time.Second

	debounceDuration = 150 * time.Millisecond
	baseMouseStep    = 10.0 // Базовый шаг мыши (начальная скорость)
)

type cursorPoint struct{ X, Y int32 }

// singleInstanceMutex держится открытым до конца процесса:
// закрытие хэндла сняло бы мьютекс.
var singleInstanceMutex windows.Handle

// --- Общее состояние (доступ из нескольких горутин) ---
var (
	stateMu         sync.Mutex
	shutdownActive  bool
	shutdownTimer   *time.Timer
	clickCount      int
	lastMouseAction string
	lastSignalTime  time.Time

	debugMode  atomic.Bool
	reloadChan = make(chan bool, 1)
)

// Соответствие имён кнопок из config.toml кодам клавиш.
// Вынесено на уровень пакета: раньше карта пересоздавалась
// на каждое нажатие.
var vkCodes = map[string]int{
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

func main() {
	checkSingleInstance()

	// Самораспаковка конфига
	if _, err := os.Stat("config.toml"); errors.Is(err, os.ErrNotExist) {
		_ = os.WriteFile("config.toml", defaultToml, 0644)
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
				// Нативный системный звук без запуска внешних
				// процессов: powershell давал вспышку окна консоли.
				playSystemSound(windows.MB_ICONASTERISK)
				// Неблокирующая отправка: повторные клики во время
				// перезагрузки просто схлопываются в одну.
				select {
				case reloadChan <- true:
				default:
				}
			case <-mDebug.ClickedCh:
				if mDebug.Checked() {
					mDebug.Uncheck()
					debugMode.Store(false)
				} else {
					mDebug.Check()
					debugMode.Store(true)
				}
			case <-mOpenConfig.ClickedCh:
				_ = exec.Command("notepad.exe", "config.toml").Start()
			case <-mQuit.ClickedCh:
				systray.Quit()
			}
		}
	}()

	go worker()
}

func worker() {
	for {
		stop := make(chan struct{})
		done := make(chan struct{})
		go runRemoteControl(stop, done)

		<-reloadChan
		close(stop)
		<-done // ждём завершения сессии вместо Sleep
	}
}

func runRemoteControl(stop <-chan struct{}, done chan<- struct{}) {
	defer close(done)

	var config struct {
		Port    string            `toml:"port"`
		Baud    int               `toml:"baud"`
		Buttons map[string]string `toml:"buttons"`
	}

	if _, err := toml.DecodeFile("config.toml", &config); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			showDebugBox("Ошибка", "Файл config.toml не найден")
		} else {
			showDebugBox("Ошибка синтаксиса TOML", err.Error())
		}
		return
	}
	sPort, err := serial.OpenPort(&serial.Config{Name: config.Port, Baud: config.Baud})
	if err != nil {
		showDebugBox("Ошибка порта", "Не удалось открыть "+config.Port)
		return
	}
	// Порт закрывается ровно один раз: либо по остановке,
	// либо при выходе из функции.
	var closeOnce sync.Once
	closePort := func() {
		closeOnce.Do(func() { _ = sPort.Close() })
	}
	defer closePort()

	go func() {
		<-stop
		closePort()
	}()

	kb, err := keybd_event.NewKeyBonding()
	if err != nil {
		showDebugBox("Ошибка клавиатуры", err.Error())
		return
	}
	scanner := bufio.NewScanner(sPort)

	var lastProcessedTime time.Time

	for scanner.Scan() {
		msg := strings.TrimSpace(scanner.Text())
		if msg == "" {
			continue
		}
		if debugMode.Load() {
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

		handleAction(action, &kb)
	}

	if err := scanner.Err(); err != nil {
		select {
		case <-stop:
			// Штатная остановка по перезагрузке конфига — не шуметь.
		default:
			showDebugBox("Ошибка чтения порта", err.Error())
		}
	}
}

// handleAction принимает указатель: методы KeyBonding меняют
// состояние получателя, копирование структуры здесь ни к чему.
func handleAction(action string, kb *keybd_event.KeyBonding) {
	// 1. Проверка: если процесс выключения уже запущен, любое действие его отменяет
	if isShuttingDown() {
		cancelShutdown()
		return
	}

	switch action {
	case "SYSTEM_SHUTDOWN":
		startShutdown()
	case "MOUSE_UP":
		moveMouseAccel("UP", 0, -1)
	case "MOUSE_DOWN":
		moveMouseAccel("DOWN", 0, 1)
	case "MOUSE_LEFT":
		moveMouseAccel("LEFT", -1, 0)
	case "MOUSE_RIGHT":
		moveMouseAccel("RIGHT", 1, 0)
	case "L_CLICK":
		mouseLeftClick()
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
	case "ALT_F4":
		kb.HasALT(true)               // Зажать Alt
		kb.SetKeys(keybd_event.VK_F4) // Выбрать F4
		kb.Launching()
		kb.HasALT(false) // ОТПУСТИТЬ Alt (Обязательно!)
	default:
		if vkCode, found := vkCodes[action]; found {
			kb.SetKeys(vkCode)
			kb.Launching()
		}
	}
}

func isShuttingDown() bool {
	stateMu.Lock()
	defer stateMu.Unlock()
	return shutdownActive
}

func startShutdown() {
	stateMu.Lock()
	if shutdownTimer != nil {
		shutdownTimer.Stop()
	}
	shutdownActive = true
	shutdownTimer = time.AfterFunc(shutdownTimeout, func() {
		shutdownPC()
	})
	stateMu.Unlock()

	playSystemSound(windows.MB_ICONEXCLAMATION)

	// 2. Окно ожидания
	go func() {
		t, err := windows.UTF16PtrFromString("Компьютер выключится через 10 секунд. Нажмите любую кнопку на пульте или закройте это окно для отмены.")
		if err != nil {
			return
		}
		c, err := windows.UTF16PtrFromString("Внимание")
		if err != nil {
			return
		}

		// MessageBox блокируется, пока окно не закроют
		_, _ = windows.MessageBox(0, t, c, windows.MB_OK|windows.MB_ICONINFORMATION|windows.MB_TOPMOST)

		// Как только окно закрыли — проверяем, активно ли еще выключение
		if isShuttingDown() {
			cancelShutdown()
		}
	}()
}

func cancelShutdown() {
	stateMu.Lock()
	if !shutdownActive {
		stateMu.Unlock()
		return
	}
	if shutdownTimer != nil {
		shutdownTimer.Stop()
		shutdownTimer = nil
	}
	shutdownActive = false
	stateMu.Unlock()

	playSystemSound(beepSimple)
}

func showDebugBox(title, text string) {
	t, err := windows.UTF16PtrFromString(text)
	if err != nil {
		return
	}
	c, err := windows.UTF16PtrFromString(title)
	if err != nil {
		return
	}
	go func() {
		_, _ = windows.MessageBox(0, t, c, windows.MB_OK|windows.MB_ICONINFORMATION|windows.MB_TOPMOST)
	}()
}

func moveMouseAccel(dir string, dx, dy int32) {
	now := time.Now()

	stateMu.Lock()
	// 1. Сброс при долгой паузе (160мс) или смене направления
	if now.Sub(lastSignalTime) > 160*time.Millisecond || dir != lastMouseAction {
		clickCount = 1
		lastMouseAction = dir
	} else {
		clickCount++
	}
	n := clickCount
	stateMu.Unlock()

	var acceleration float64
	// 3. Условие: больше трех нажатий (т.е. начиная с 4-го)
	if n > 3 {
		// Рассчитываем ускорение на основе количества накопленных сигналов
		// Чем чаще приходят сигналы, тем выше скорость
		acceleration = 60.0 * math.Log1p(float64(n-3))
	}

	// Итоговый шаг
	step := int32(baseMouseStep + acceleration)

	pt, err := getCursorPos()
	if err != nil {
		return
	}

	_ = setCursorPos(
		pt.X+dx*step,
		pt.Y+dy*step,
	)

	// Обновляем время последнего сигнала в самом конце
	stateMu.Lock()
	lastSignalTime = now
	stateMu.Unlock()
}

func checkSingleInstance() {
	mutexName, err := windows.UTF16PtrFromString(`Local\IRRemote`)
	if err != nil {
		return
	}
	h, err := windows.CreateMutex(nil, true, mutexName)
	if h == 0 {
		return
	}
	if errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		os.Exit(0)
	}
	singleInstanceMutex = h
}

func onExit() {}

// --- Низкоуровневые обёртки user32 (нет в x/sys/windows) ---

func getCursorPos() (cursorPoint, error) {
	var pt cursorPoint
	r1, _, e1 := procGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	if r1 == 0 {
		return pt, e1
	}
	return pt, nil
}

func setCursorPos(x, y int32) error {
	r1, _, e1 := procSetCursorPos.Call(uintptr(x), uintptr(y))
	if r1 == 0 {
		return e1
	}
	return nil
}

func mouseLeftClick() {
	// mouse_event возвращает void — проверять нечего.
	_, _, _ = procMouseEvent.Call(mouseLeftDown, 0, 0, 0, 0)
	_, _, _ = procMouseEvent.Call(mouseLeftUp, 0, 0, 0, 0)
}

// Функция чистого выключения
func shutdownPC() {
	// Сначала получаем права
	if err := getShutdownPrivileges(); err != nil {
		showDebugBox("Ошибка выключения", err.Error())
		return
	}

	// Вызываем API выключения
	// EWX_SHUTDOWN | EWX_POWEROFF — выключить питание
	// EWX_FORCE — закрыть приложения без вопросов
	if err := windows.ExitWindowsEx(windows.EWX_SHUTDOWN|windows.EWX_POWEROFF|windows.EWX_FORCE, 0); err != nil {
		showDebugBox("Ошибка выключения", err.Error())
	}
}

func getShutdownPrivileges() error {
	hProcess, err := windows.GetCurrentProcess()
	if err != nil {
		return fmt.Errorf("не удалось получить текущий процесс: %w", err)
	}

	// 1. Открываем токен текущего процесса
	var hToken windows.Token
	if err := windows.OpenProcessToken(hProcess, windows.TOKEN_ADJUST_PRIVILEGES|windows.TOKEN_QUERY, &hToken); err != nil {
		return fmt.Errorf("не удалось открыть токен процесса: %w", err)
	}
	defer windows.CloseHandle(windows.Handle(hToken))

	// 2. Находим LUID для привилегии выключения
	var tkp windows.Tokenprivileges
	namePtr, err := windows.UTF16PtrFromString(shutdownPrivilege)
	if err != nil {
		return fmt.Errorf("не удалось закодировать имя привилегии: %w", err)
	}
	if err := windows.LookupPrivilegeValue(nil, namePtr, &tkp.Privileges[0].Luid); err != nil {
		return fmt.Errorf("не удалось найти привилегию: %w", err)
	}

	tkp.PrivilegeCount = 1
	tkp.Privileges[0].Attributes = windows.SE_PRIVILEGE_ENABLED

	// 3. Применяем привилегию к нашему процессу
	if err := windows.AdjustTokenPrivileges(hToken, false, &tkp, 0, nil, nil); err != nil {
		return fmt.Errorf("не удалось применить привилегии: %w", err)
	}

	return nil
}

func playSystemSound(soundType uint32) {
	// Вызов WinAPI функции напрямую
	_, _, _ = procMessageBeep.Call(uintptr(soundType))
}
