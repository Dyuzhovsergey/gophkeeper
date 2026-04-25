# GophKeeper

GophKeeper — клиент-серверный менеджер приватных данных на Go.

Проект позволяет:
- регистрировать и аутентифицировать пользователей;
- хранить приватные данные разных типов;
- получать список и детали секретов;
- работать с CLI/TUI-клиентом.

Поддерживаемые типы данных:
- логин/пароль;
- текстовые данные;
- банковские карты;
- бинарные данные.

Для всех типов данных поддерживается произвольная текстовая метаинформация.
Для binary-данных клиент использует файл как источник бинарного содержимого.

## Текущий стек

- Go
- PostgreSQL
- HTTP/JSON
- JWT для авторизации
- Bubble Tea для TUI-клиента


## Сервер
- регистрация пользователя;
- login пользователя;
- JWT-аутентификация;
- CRUD для секретов;
- хранение данных в PostgreSQL;
- автоматический запуск миграций при старте;
- health endpoint.

## Клиент
- запуск в CLI/TUI;
- register / login / logout;
- сохранение локальной сессии;
- повторное использование сохранённого токена;
- просмотр текущей сессии (me);
- список секретов;
- просмотр одного секрета;
- создание, обновление и удаление секретов;
- работа с типами:
    - text
    - credentials
    - card
    - binary


## Требования
- Go 1.25.x
- PostgreSQL


## Подготовка PostgreSQL
Нужно создать базу данных и пользователя, затем убедиться, что доступ по DSN работает.
Пример DSN:

```bash
postgres://user:YOUR_PASSWORD@localhost:5432/gophkeeper?sslmode=disable
```
## Проверка подключения:
```bash
psql -h localhost -p 5432 -U user -d gophkeeper
```

## Конфигурация сервера

Сервер читает конфигурацию из:
- значений по умолчанию;
- флагов;
- переменных окружения.


## Основные переменные окружения
- GOPHKEEPER_SERVER_RUN_ADDRESS
- GOPHKEEPER_SERVER_LOG_LEVEL
- GOPHKEEPER_SERVER_DATABASE_DSN
- GOPHKEEPER_SERVER_JWT_SECRET

## Основные флаги
-a           # адрес запуска HTTP-сервера
-log-level   # уровень логирования
-d           # PostgreSQL DSN
-jwt-secret  # секрет подписи JWT



## Запуск сервера

Пример через переменные окружения:
```bash
export GOPHKEEPER_SERVER_RUN_ADDRESS=localhost:8080
export GOPHKEEPER_SERVER_LOG_LEVEL=debug
export GOPHKEEPER_SERVER_DATABASE_DSN='postgres://user:YOUR_PASSWORD@localhost:5432/gophkeeper?sslmode=disable'
export GOPHKEEPER_SERVER_JWT_SECRET='very-secret-key'

go run ./cmd/server
```

## Пример через флаги:
```bash
go run ./cmd/server \
  -a localhost:8080 \
  -log-level debug \
  -d 'postgres://user:YOUR_PASSWORD@localhost:5432/gophkeeper?sslmode=disable' \
  -jwt-secret 'very-secret-key'
  ```


## Проверка health endpoint
```bash
curl -i http://localhost:8080/health
```
Ожидается HTTP 200.


## Запуск клиента
TUI-режим
```bash
go run ./cmd/client
```

## Режим CLI-команды
```bash
go run ./cmd/client version
```
```bash
go run ./cmd/client register <login> <password>
```
```bash
go run ./cmd/client login <login> <password>
```
```bash
go run ./cmd/client logout
```


## Работа с клиентом
Базовый сценарий нового пользователя

1. Запустить сервер.
2. Запустить клиент.
3. Зарегистрироваться.
4. Выполнить login.
5. Добавить секреты через TUI.
6. Открыть secrets и проверить список.
6. Открыть нужный секрет и посмотреть детали.


## Локальная сессия клиента

После успешного login клиент сохраняет локальную сессию.
При следующем запуске клиент может использовать сохранённый токен без повторного ввода логина и пароля, пока серверная сессия остаётся валидной.
Локально также может храниться время последней успешной синхронизации.


## Возможности TUI

В TUI доступны пункты меню:
- register
- login
- logout
- me
- secrets
- sync
- add text
- add credentials
- add card
- add file
- version
- quit

В TUI можно:
- просматривать список секретов;
- открывать один секрет;
- создавать секреты всех обязательных типов;
- редактировать секреты;
- удалять секреты с подтверждением;
- выполнять sync с сервером.


## Sync

Сервер является источником истины.

Клиент поддерживает sync изменений между несколькими авторизованными клиентами одного владельца.

Что делает sync:
- получает изменения после времени последней синхронизации;
- получает и обычные изменения, и удалённые записи;
- сохраняет новое время последней синхронизации локально.

Конфликтная политика:
- last write wins;
- итоговое состояние определяется последним успешным изменением на сервере.



## Сборка
Сервер
```bash
go build -o bin/server ./cmd/server
```
Клиент
```bash
go build -o bin/client ./cmd/client
```

## Тесты

Запуск всех тестов:
```bash
go test ./...
```