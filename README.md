# Chat System with Task Management

Комплексная система чата с интегрированным таск-трекером, help desk функциональностью и поддержкой команд.

## 🚀 Основные возможности

- **Real-time чат** с поддержкой групп и direct messages
- **Система команд** для управления задачами прямо из чата
- **Task management** с state machine для статусов
- **Help Desk** функциональность с SLA tracking
- **Keycloak интеграция** для SSO и управления пользователями
- **HTMX + Alpine.js** для минимального использования JavaScript
- **WebSocket/SSE** для real-time обновлений
- **Плагины** для расширения функциональности

## 📋 Содержание документации

- [Архитектура](./docs/01-architecture.md) - Общая архитектура системы
- [Установка и настройка](./docs/02-installation.md) - Руководство по установке
- [База данных](./docs/03-database.md) - Схема БД и миграции
- [Backend разработка](./docs/04-backend.md) - Go сервисы и API
- [Frontend с HTMX](./docs/05-frontend-htmx.md) - HTMX templates и компоненты
- [Keycloak интеграция](./docs/06-keycloak.md) - SSO и аутентификация
- [WebSocket/Real-time](./docs/07-websocket.md) - Real-time функциональность
- [Система команд](./docs/08-commands.md) - Command parser и handlers
- [Help Desk](./docs/09-helpdesk.md) - SLA и support функции
- [Плагины](./docs/10-plugins.md) - Система плагинов
- [Тестирование](./docs/11-testing.md) - Unit, integration и E2E тесты
- [Deployment](./docs/12-deployment.md) - Production deployment
- [Мониторинг](./docs/13-monitoring.md) - Метрики и health checks
- [API документация](./docs/14-api.md) - REST API endpoints

## 🛠 Технологический стек

### Backend
- **Go 1.25+** - основной язык
- **Echo v4** - веб-фреймворк
- **PostgreSQL 18+** - основная БД
- **Redis** - кеш и pub/sub
- **Keycloak** - SSO и управление пользователями

### Frontend
- **HTMX 2+** - динамические обновления
- **Pico CSS v2** - минималистичный CSS фреймворк

## 📁 Структура проекта

```
chat-system/
├── cmd/                    # Точки входа приложений
│   ├── api/               # HTTP API сервер
│   ├── websocket/         # WebSocket сервер
│   ├── worker/            # Background workers
│   └── migrator/          # DB миграции
├── internal/              # Внутренний код приложения
│   ├── domain/           # Бизнес-логика и модели
│   ├── repository/       # Слой работы с БД
│   ├── service/          # Сервисный слой
│   ├── handler/          # HTTP/WS handlers
│   ├── auth/             # Аутентификация
│   │   └── keycloak/     # Keycloak клиент
│   ├── command/          # Command processors
│   └── event/            # Event bus
├── pkg/                   # Переиспользуемые пакеты
├── web/                   # Frontend ресурсы
│   ├── templates/        # HTML templates
│   ├── static/           # CSS, JS
│   └── components/       # HTMX компоненты
├── migrations/           # SQL миграции
├── config/               # Конфигурационные файлы
├── deployments/          # Docker и K8s конфиги
├── scripts/              # Utility scripts
├── tests/                # Тесты
└── docs/                 # Документация
```

## 🚦 Quick Start

### Требования

- Go 1.25+
- Docker & Docker Compose
- PostgreSQL 17+
- Redis 7+

### Локальная разработка

1. Клонировать репозиторий:
```bash
git clone https://github.com/yourorg/chat-system.git
cd chat-system
```

2. Скопировать конфигурацию:
```bash
cp .env.example .env
# Отредактировать .env файл
```

3. Запустить инфраструктуру:
```bash
docker-compose up -d postgres redis keycloak
```

4. Выполнить миграции:
```bash
go run cmd/migrator/main.go up
```

5. Запустить приложение:
```bash
go run cmd/api/main.go
```

Приложение будет доступно на http://localhost:8080

## 📊 Timeline проекта

### Фазы разработки

| Фаза | Описание | Длительность | Статус |
|------|----------|--------------|--------|
| 1 | Подготовка и базовая архитектура | 2-3 недели | 🔄 |
| 2 | Базовая функциональность | 3-4 недели | ⏳ |
| 3 | WebSocket и Real-time | 2-3 недели | ⏳ |
| 4 | Система команд и задач | 3-4 недели | ⏳ |
| 5 | Help Desk функциональность | 2-3 недели | ⏳ |
| 6 | UI и пользовательский опыт | 2-3 недели | ⏳ |
| 7 | Background Jobs | 2 недели | ⏳ |
| 8 | Деплой и мониторинг | 2 недели | ⏳ |
| 9 | Оптимизация | 2-3 недели | ⏳ |
| 10 | Тестирование | 2-3 недели | ⏳ |
| 11 | Расширенные функции | 3-4 недели | ⏳ |
| 12 | Безопасность | 2 недели | ⏳ |
| 13 | Документация | 1-2 недели | ⏳ |

**Общее время**: 25-30 недель (6-7 месяцев)
**MVP**: 10-12 недель

## 🔐 Безопасность

- OAuth 2.0/OIDC через Keycloak
- JWT tokens с refresh механизмом
- RBAC (Role-Based Access Control)
- Rate limiting
- CORS защита
- SQL injection защита через prepared statements
- XSS защита через template escaping
- CSRF токены для форм

## 🧪 Тестирование

```bash
# Unit тесты
go test ./...

# Integration тесты
go test ./tests/integration -tags=integration

# E2E тесты
go test ./tests/e2e -tags=e2e

# Нагрузочное тестирование
go test ./tests/load -tags=load
```

## 📈 Мониторинг

- Prometheus метрики на `/metrics`
- Health checks на `/health`
- Grafana дашборды
- Structured logging через zerolog
- Distributed tracing (опционально)

## 🤝 Contributing

См. [CONTRIBUTING.md](./CONTRIBUTING.md) для деталей.

## 📄 Лицензия

[MIT License](./LICENSE)

## 📞 Поддержка

- Email: support@yourcompany.com
- Slack: #chat-system-dev
- Issues: GitHub Issues

## 🙏 Благодарности

- [HTMX](https://htmx.org/) - за минималистичный подход к динамическому HTML
- [Echo](https://echo.labstack.com/) - за быстрый веб-фреймворк
- [Keycloak](https://www.keycloak.org/) - за мощную систему аутентификации

---

**Version**: 1.0.0
