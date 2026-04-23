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

## Текущий стек

- Go
- PostgreSQL
- HTTP/JSON
- JWT для авторизации
- Bubble Tea для TUI-клиента

## Структура проекта

.
├── bin
│   ├── client
│   └── server
├── cmd
│   ├── client
│   │   └── main.go
│   └── server
│       └── main.go
├── docs
│   └── api-outline.md
├── dsn.md
├── go.mod
├── go.sum
├── internal
│   ├── buildinfo
│   │   ├── buildinfo.go
│   │   └── buildinfo_test.go
│   ├── cli
│   │   ├── commands
│   │   │   ├── root.go
│   │   │   └── version.go
│   │   ├── doc.go
│   │   ├── model.go
│   │   ├── model_test.go
│   │   └── output
│   ├── clientapi
│   │   ├── auth.go
│   │   ├── clientapi_test.go
│   │   ├── client.go
│   │   ├── doc.go
│   │   └── secrets.go
│   ├── config
│   │   ├── client.go
│   │   ├── common.go
│   │   ├── config_test.go
│   │   └── server.go
│   ├── domain
│   │   ├── auth.go
│   │   ├── errors.go
│   │   ├── secret.go
│   │   └── user.go
│   ├── logger
│   │   ├── logger.go
│   │   └── logger_test.go
│   ├── repository
│   │   ├── interfaces.go
│   │   └── postgres
│   │       ├── postgres.go
│   │       ├── secrets.go
│   │       ├── secrets_test.go
│   │       ├── sessions.go
│   │       ├── sessions_test.go
│   │       ├── users.go
│   │       └── users_test.go
│   ├── security
│   │   ├── bcrypt_test.go
│   │   ├── jwt_test.go
│   │   ├── password.go
│   │   └── token.go
│   ├── service
│   │   ├── auth
│   │   │   ├── doc.go
│   │   │   ├── service.go
│   │   │   └── service_test.go
│   │   └── vault
│   │       ├── doc.go
│   │       ├── service.go
│   │       └── service_test.go
│   ├── storage
│   │   └── local
│   │       ├── doc.go
│   │       ├── session.go
│   │       ├── store.go
│   │       └── store_test.go
│   └── transport
│       └── http
│           ├── dto
│           │   ├── auth.go
│           │   └── secrets.go
│           ├── handlers
│           │   ├── auth.go
│           │   ├── auth_test.go
│           │   ├── health.go
│           │   ├── secrets.go
│           │   └── secrets_test.go
│           ├── middleware
│           │   ├── auth.go
│           │   ├── auth_test.go
│           │   ├── context.go
│           │   └── doc.go
│           ├── response
│           │   ├── doc.go
│           │   ├── response.go
│           │   └── response_test.go
│           ├── router.go
│           └── router_test.go
├── LICENSE
├── Makefile
├── migrations
│   └── migrations.go
└── README.md



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

## Подготовка PostgreSQL
Нужно создать базу данных и пользователя, затем убедиться, что доступ по DSN работает.
Пример DSN:

```
postgres://user:YOUR_PASSWORD@localhost:5432/gophkeeper?sslmode=disable
```

## Проверка подключения:

psql -h localhost -p 5432 -U user -d gophkeeper


## Запуск сервера

Пример через переменные окружения:

export GOPHKEEPER_SERVER_RUN_ADDRESS=localhost:8080
export GOPHKEEPER_SERVER_LOG_LEVEL=debug
export GOPHKEEPER_SERVER_DATABASE_DSN='postgres://user:YOUR_PASSWORD@localhost:5432/gophkeeper?sslmode=disable'
export GOPHKEEPER_SERVER_JWT_SECRET='very-secret-key'

go run ./cmd/server


## Пример через флаги:

go run ./cmd/server \
  -a localhost:8080 \
  -log-level debug \
  -d 'postgres://user:YOUR_PASSWORD@localhost:5432/gophkeeper?sslmode=disable' \
  -jwt-secret 'very-secret-key'