# Changelog

Все заметные изменения проекта.

Формат — [Keep a Changelog](https://keepachangelog.com/ru/1.1.0/),
версии соответствуют тегам и релизам GitHub.

## [Unreleased]

### Изменено
- README.md и README.en.md освежены (переписаны по re0)
- config.toml: обновлены комментарии и справочник команд (бинды кнопок не менялись)

## [0.1.0] — 2026-10-21

### Добавлено
- Единый скетч `IR-controler-dual.ino` с автоопределением протокола пульта
  (NEC/RC5) — один скетч подходит для большинства пультов
- Префиксы кодов кнопок в config: `NEC_xx` / `RC5_xx`
  (оба пульта работают одновременно)
- Английская версия README — `README.en.md`
- Раздача exe через GitHub Releases

### Изменено
- README полностью переписан (русская версия)
- `config.toml`: набор кодов `RC5_*` и пример `NEC_2`

### Удалено
- Однопротокольные скетчи `IR-controler-nec.ino` и `IR-controler-rc5.ino`
- Библиотека NecDecoder

## [0.0.5] — 2026-09-14

Первая версия.

- Управление ПК ИК-пультом через Arduino (COM-порт, 115200)
- Эмуляция клавиатуры
- «Умная» мышь с ускорением
- Выключение ПК с 10-секундным таймером отмены
- Работа в системном трее
- Раздельные скетчи под протоколы NEC и RC5

[Unreleased]: https://github.com/Rocaccel/ir-comp-control/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/Rocaccel/ir-comp-control/compare/v0.0.5...v0.1.0
[0.0.5]: https://github.com/Rocaccel/ir-comp-control/releases/tag/v0.0.5
