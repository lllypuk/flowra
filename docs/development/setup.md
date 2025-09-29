# Руководство по настройке среды разработки

## Обзор

Этот документ содержит инструкции по настройке локальной среды разработки для проекта New Teams Up.

## Системные требования

### Обязательные компоненты

- **Go**: версия 1.19 или выше
- **PostgreSQL**: версия 14 или выше
- **Redis**: версия 6 или выше
- **Docker**: версия 20.10 или выше
- **Docker Compose**: версия 2.0 или выше
- **Git**: версия 2.30 или выше

### Рекомендуемые инструменты

- **Make**: для автоматизации задач
- **golangci-lint**: для статического анализа кода
- **Air**: для hot reload в development
- **Postman** или **Insomnia**: для тестирования API
- **pgAdmin** или **DBeaver**: для работы с базой данных

## Установка зависимостей

### macOS (с использованием Homebrew)

```bash
# Go
brew install go

# PostgreSQL
brew install postgresql@14
brew services start postgresql@14

# Redis
brew install redis
brew services start redis

# Docker
brew install --cask docker

# Дополнительные инструменты
brew install make
brew install golangci/tap/golangci-lint
```

### Ubuntu/Debian

```bash
# Go
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# PostgreSQL
sudo apt update
sudo apt install postgresql postgresql-contrib
sudo systemctl start postgresql
sudo systemctl enable postgresql

# Redis
sudo apt install redis-server
sudo systemctl start redis-server
sudo systemctl enable redis-server

# Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# Дополнительные инструменты
sudo apt install make
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin
```

### Windows

1. Установите Go с официального сайта: https://golang.org/dl/
2. Установите PostgreSQL: https://www.postgresql.org/download/windows/
3. Установите Redis: https://redis.io/download
4. Установите Docker Desktop: https://www.docker.com/products/docker-desktop

## Настройка проекта

### 1. Клонирование репозитория

```bash
git clone https://github.com/lllypuk/new-teams-up.git
cd new-teams-up
```

### 2. Настройка переменных окружения

Создайте файл `.env` в корне проекта:

```bash
cp .env.example .env
```

Отредактируйте `.env` файл:

```env
# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=new_teams_up_dev
DB_USER=postgres
DB_PASSWORD=your_password
DB_SSL_MODE=disable

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# JWT
JWT_SECRET=your-super-secret-jwt-key-here
JWT_EXPIRES_IN=24h

# Server
SERVER_HOST=localhost
SERVER_PORT=8080

# Environment
ENV=development
LOG_LEVEL=debug

# External Services
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USER=your-email@gmail.com
SMTP_PASSWORD=your-app-password

# OAuth
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret
```

### 3. Настройка базы данных

Создайте базу данных PostgreSQL:

```bash
# Подключитесь к PostgreSQL
psql -U postgres

# Создайте пользователя и базу данных
CREATE USER new_teams_up WITH PASSWORD 'your_password';
CREATE DATABASE new_teams_up_dev OWNER new_teams_up;
CREATE DATABASE new_teams_up_test OWNER new_teams_up;
GRANT ALL PRIVILEGES ON DATABASE new_teams_up_dev TO new_teams_up;
GRANT ALL PRIVILEGES ON DATABASE new_teams_up_test TO new_teams_up;

# Выйдите из psql
\q
```

### 4. Установка Go зависимостей

```bash
go mod download
go mod tidy
```

### 5. Запуск миграций

```bash
# Установите migrate tool
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Запустите миграции
make migrate-up
```

### 6. Заполнение тестовыми данными (опционально)

```bash
make seed
```

## Запуск приложения

### Development режим

```bash
# Установите Air для hot reload
go install github.com/cosmtrek/air@latest

# Запустите в development режиме
make dev
```

### Production режим

```bash
# Соберите приложение
make build

# Запустите скомпилированный бинарник
make run
```

### Docker Compose

```bash
# Запустите все сервисы в Docker
make docker-up

# Остановите сервисы
make docker-down
```

## Доступные команды Make

```bash
# Разработка
make dev          # Запуск в development режиме с hot reload
make build        # Сборка приложения
make run          # Запуск приложения
make clean        # Очистка build файлов

# Тестирование
make test         # Запуск всех тестов
make test-unit    # Запуск unit тестов
make test-integration # Запуск integration тестов
make coverage     # Генерация отчета о покрытии тестами

# Качество кода
make lint         # Проверка кода линтером
make fmt          # Форматирование кода
make vet          # Проверка кода vet'ом

# База данных
make migrate-up   # Применить миграции
make migrate-down # Откатить миграции
make migrate-create # Создать новую миграцию
make seed         # Заполнить БД тестовыми данными

# Docker
make docker-build # Собрать Docker образ
make docker-up    # Запустить сервисы в Docker
make docker-down  # Остановить Docker сервисы

# Документация
make docs         # Генерация API документации
make swagger      # Запуск Swagger UI
```

## Структура проекта

После настройки у вас должна быть следующая структура:

```
new-teams-up/
├── .env                    # Переменные окружения (не коммитится)
├── .air.toml              # Конфигурация Air
├── docker-compose.yml     # Docker Compose конфигурация
├── Makefile              # Автоматизация задач
├── cmd/                  # Точки входа приложений
├── internal/             # Внутренний код приложения
├── pkg/                  # Переиспользуемые пакеты
├── migrations/           # Миграции базы данных
├── configs/              # Конфигурационные файлы
├── scripts/              # Вспомогательные скрипты
└── docs/                 # Документация
```

## Настройка IDE

### VS Code

Рекомендуемые расширения:

```json
{
  "recommendations": [
    "golang.go",
    "ms-vscode-remote.remote-containers",
    "ms-azuretools.vscode-docker",
    "ms-vscode.vscode-json",
    "redhat.vscode-yaml",
    "bradlc.vscode-tailwindcss"
  ]
}
```

Настройки VS Code (`.vscode/settings.json`):

```json
{
  "go.formatTool": "gofmt",
  "go.lintTool": "golangci-lint",
  "go.vetOnSave": "package",
  "go.buildOnSave": "package",
  "go.testFlags": ["-v", "-race"],
  "go.coverOnSave": true,
  "go.coverageOptions": "showUncoveredCodeOnly",
  "files.exclude": {
    "**/.git": true,
    "**/node_modules": true,
    "**/vendor": true
  }
}
```

### GoLand

1. Откройте проект в GoLand
2. Настройте Go Modules в Settings → Go → Go Modules
3. Настройте Database connection в Database panel
4. Установите плагины: Docker, Database Tools and SQL

## Отладка

### Delve Debugger

```bash
# Установите delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Запустите с отладчиком
dlv debug ./cmd/api
```

### VS Code Debugging

Конфигурация `.vscode/launch.json`:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Launch API",
      "type": "go",
      "request": "launch",
      "mode": "auto",
      "program": "${workspaceFolder}/cmd/api",
      "env": {
        "ENV": "development"
      },
      "args": []
    }
  ]
}
```

## Тестирование

### Unit тесты

```bash
# Запуск всех unit тестов
go test ./...

# Запуск тестов с покрытием
go test -cover ./...

# Детальный отчет о покрытии
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Integration тесты

```bash
# Запуск integration тестов
go test -tags=integration ./...
```

### Benchmarks

```bash
# Запуск бенчмарков
go test -bench=. ./...
```

## Troubleshooting

### Частые проблемы

**1. Не удается подключиться к PostgreSQL**
```bash
# Проверьте статус сервиса
sudo systemctl status postgresql

# Проверьте настройки в pg_hba.conf
sudo nano /etc/postgresql/14/main/pg_hba.conf
```

**2. Go модули не загружаются**
```bash
# Очистите модульный кэш
go clean -modcache

# Переустановите зависимости
go mod download
```

**3. Port уже используется**
```bash
# Найдите процесс, использующий порт
lsof -i :8080

# Завершите процесс
kill -9 <PID>
```

**4. Docker проблемы**
```bash
# Очистите Docker контейнеры и образы
docker system prune -a

# Перезапустите Docker daemon
sudo systemctl restart docker
```

## Следующие шаги

После успешной настройки среды разработки:

1. Изучите [стандарты кодирования](coding-standards.md)
2. Ознакомьтесь с [стратегией тестирования](testing.md)
3. Прочитайте [архитектурное описание](../../ARCHITECTURE.md)
4. Начните с изучения [API документации](../api/)

## Помощь

Если у вас возникли проблемы:

1. Проверьте [FAQ](../faq.md)
2. Поищите решение в [Issues](https://github.com/lllypuk/new-teams-up/issues)
3. Создайте новый Issue с тегом `setup`
4. Обратитесь в Slack канал `#development`

---

*Последнее обновление: [Текущая дата]*
*Поддерживается: Development Team*
