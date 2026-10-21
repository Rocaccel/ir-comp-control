# AGENTS.md — правила работы с репозиторием

Проект: **IR_comp_control** — управление Windows-ПК ИК-пультом через Arduino
(ИК-приёмник на пин D2) и Go-приложение в системном трее.

## Git

- **Коммиты** — conventional commits: тип и scope по-английски,
  описание по-русски. Примеры:
  - `feat(ir): ...`
  - `fix(mouse): ...`
  - `docs(readme): ...`
  - `chore: ...`
- **PUSH ТОЛЬКО ПО ЯВНОМУ УКАЗАНИЮ ПОЛЬЗОВАТЕЛЯ.**
  Никогда не пушить по своей инициативе — после коммита спросить.
- Не force-push. Основная ветка — `main`.
- Фичи — в отдельных ветках; при большой работе — через worktree.
- Мерж в `main` — с merge-коммитом:
  `git merge --no-ff <ветка> -m "Merge branch 'v0.1.0': краткое содержание"`
- Ветки после мержа не удалять (история: `v0.0.5`, `v0.1.0` остаются).
- Перед коммитом/мержем — `git status`; при сомнениях — `git fetch` + `git status`.

### Рабочие папки (на этой машине)

- `C:\Users\ser\Desktop\GitHub\IR_comp_control` — основная копия, ветка `main`
- `C:\Users\ser\Desktop\GOLANG_Portable\workspace\IR-control-v0.1.0` —
  worktree ветки `v0.1.0` (скетчи, сборка exe)
- `C:\Users\ser\Desktop\GOLANG_Portable\workspace\IR-control-v0.0.5` —
  архив старой версии, не трогать

## Репозиторий

- `.gitignore`: `*.exe` и корневой `config.toml` — **не коммитить**.
  Exe раздаётся через Releases; корневой `config.toml` — локальный
  runtime-файл, его не публикуют (в том числе в релизах).
- README двуязычный: `README.md` (русский, канон) и `README.en.md`
  (английский) — правки вносить в оба файла и держать синхронными.
- `CHANGELOG.md` — обновлять при релизах и заметных изменениях
  (формат Keep a Changelog).

## Конфигурация

- `src_golang/config.toml` — шаблон, вшивается в exe через `//go:embed`.
- При первом запуске exe распаковывает его рядом с собой, **только если**
  `config.toml` там отсутствует; существующий файл никогда не перезаписывается.
- Коды кнопок с префиксом протокола: `NEC_XX` / `RC5_XX` — протокол
  определяется автоматически, оба пульта работают одновременно.
- Порт и скорость: `port = "COM4"`, `baud = 115200`.

## Прошивка Arduino

- Скетч: `for_arduino_controller/IR-controler-dual.ino` — один скетч,
  автоопределение NEC/RC5. Библиотека IRremote 4.x
  (в 4.5.0 нет констант `NEC_EXTRA`/`RC5X`).
- ИК-приёмник — на пин D2.
- **Имя папки при сборке = имя .ino** — IDE копирует скетч
  в `%TEMP%\<имя-файла>\`.
- Загрузка: `arduino.exe --upload --board arduino:avr:uno <файл.ino>`
  (IDE 1.8.19, порт COM4).
- Перед запуском exe — закрыть Монитор порта Arduino IDE.

## Сборка exe (Windows)

```sh
go build -trimpath -ldflags "-s -w -H=windowsgui" -o ir-comp-control_v<версия>-win64.exe .
```

### Имя файла релиза

Формируется по формуле:

```
ir-comp-control_v<версия>-win64.exe
```

- `ir-comp-control` — название проекта
- `v<версия>` — версия релиза/тега (например `v0.0.5`, `v0.1.0`)
- `win64` — платформа

Пример: `ir-comp-control_v0.1.0-win64.exe`

## Релизы

1. Тег ставится на коммит в `main`. API GitHub принимает только
   **полный SHA** (короткий отклоняется ошибкой 422).
2. Команда:

   ```sh
   gh release create v<версия> <путь-к-exe> --repo Rocaccel/ir-comp-control --target <полный-SHA> --title "v<версия>" --notes-file <файл-notes>
   ```

3. Notes — на русском: что нового и установка.
4. К релизу прикладывать **только exe** (корневой `config.toml` — нет).
5. Исторические релизы не удалять (v0.0.5 опубликован задним числом
   для истории).

## GitHub

- Описание и топики репозитория:
  `gh repo edit Rocaccel/ir-comp-control --description "..." --add-topic ...`
- Протухший токен: `gh auth refresh -h github.com`
