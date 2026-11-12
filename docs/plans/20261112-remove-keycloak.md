# Удаление Keycloak: вход только токеном

## Overview

Задача лида T-0015, часть 2. Решение владельца 12.11: «выбрасывать КК». Сейчас в коде два режима входа
(`auth.mode: keycloak | token`) и запасные пути без Keycloak: `MockAuthService`, `StaticTokenValidator`,
mock-сессия в двух middleware, `NoOpKeycloakClient`. Владелец работает в режиме `token` (`make dev-local`).

Что должно получиться:
- единственный способ входа — именованные токены владельца (`OwnerTokenValidator`), без `auth.mode`;
- нет пакета `internal/infrastructure/keycloak`, синхронизации пользователей и групп, OAuth-входа, mock-входа,
  профиля `keycloak` в `docker-compose.yml`, Keycloak в `docker-compose.prod.yml`, упоминаний в документах;
- модель «несколько именованных токенов → разные пользователи» не закрыта и не построена; чего для неё не хватает —
  записано в документации;
- тесты, завязанные на Keycloak, переведены на токены или удалены с записью, что перестало проверяться.

## Context (from discovery)

- Код Keycloak: `internal/infrastructure/keycloak/`, `internal/middleware/keycloak_adapter.go`,
  `internal/service/auth_service.go` (+ `internal/infrastructure/auth` token store только для него),
  `internal/service/noop_keycloak_client.go`, `internal/worker/user_sync.go`,
  `internal/application/workspace/keycloak_client.go` и его потребители, `cmd/api/container.go`.
- Mock-вход: `middleware/auth.go` (`MockSessionToken`, `StaticTokenValidator`),
  `handler/http/auth_middleware.go:100` (`setMockUserContext` при пустом валидаторе; `RequireValidator` закрывает
  это только в режиме токена), `MockAuthService`, mock-поток в `TemplateHandler.LoginPage`.
- Режим `token`: `middleware/token_auth.go` — все токены дают одну личность `owner`; проверка конфига требует
  loopback, владельца и хотя бы один токен. `Config.Validate` вызывается в каждом `config.Load` — его делят API,
  воркер и `cmd/tools`.
- `IsProduction()` решает по `jwt_secret` (`config.go:757`); на нём держатся запрет mock-режима и mock-проверки
  доступа в проде.
- Домен: `workspace.keycloakGroupID` и `WorkspaceCreated.KeycloakGroupID`; `user.externalID` хранится в поле
  `keycloak_id` с индексом `idx_users_keycloak_unique`.
- Тесты: `tests/integration/keycloak_*_test.go`, `tests/testutil/keycloak.go`; `tests/e2e/setup_test.go` и
  `auth_test.go` (`/auth/login`, `/auth/refresh` через `MockAuthService`); `tests/e2e/frontend` входит формой
  Keycloak на живом сервере.
- YAML конфиг читается нестрого: удалённые ключи в старых файлах игнорируются.

## Development Approach

- testing approach: Regular (код, затем тесты в той же задаче).
- каждая задача заканчивается зелёными `go build ./...`, `go vet -tags=integration,e2e ./...`, `go test ./...`.
- ветка `feat/remove-keycloak` от `fix/write-contract`, без пуша и слияния.
- `AppModeMock` (режим проводки зависимостей) не убираем: это не вход.

## Solution Overview

1. Конфиг: убрать `KeycloakConfig`, `AuthMode`, `jwt_secret`, TTL токенов. Проверки входа (токены, владелец,
   loopback, `session_ttl`) — отдельный `Config.ValidateAuth()`, его вызывает только `cmd/api`: воркеру,
   `cmd/tools` и `make reset-data` токен не нужен. `DefaultHost` — `127.0.0.1`. `config.yaml` становится тем, чем
   был `config.token.yaml`; `config.token.yaml` удалить; `make dev` = нынешний `dev-local`, `dev-local` остаётся
   псевдонимом, `dev-lite` — API без воркера на том же конфиге.
2. `IsProduction()` убрать вместе с `ErrMockModeInProd`: признак держался на `jwt_secret`. Mock-проверка доступа
   разрешена только при `app.mode: mock`; `getEnvironment` — `development` при уровне журнала `debug`, иначе
   `production`.
3. Вход: `TokenValidator` — только `OwnerTokenValidator`; без него старт останавливается. Удалить
   `StaticTokenValidator`, `MockSessionToken`, `setMockUserContext`, `RequireValidator` (запрос без валидатора
   отклоняется всегда), `KeycloakValidatorAdapter`, `MockAuthService`, `DisabledAuthService`, `AuthService`, token
   store, OAuth в `TemplateHandler` и `/auth/callback`. API: `/api/v1/auth/me` остаётся; `login`, `refresh`,
   `logout` удаляются — сервер не выдаёт и не отзывает Bearer-токены; веб-выход — `POST /auth/logout` (кука).
4. Workspace (сделано): без `keycloakGroupID`, `KeycloakClient`, `FindByKeycloakGroup`, индекса групп;
   `ListUserWorkspacesUseCase` (неподключённая заглушка) удалён, остаётся убрать `ListUserWorkspacesQuery`.
5. Пользователи: `externalID` остаётся — ключ личности из токена. Поле хранения `keycloak_id` и имя индекса
   остаются (переименование — миграция базы владельца без выгоды); Go-поле документа — `ExternalID` с тегом
   `bson:"keycloak_id"`. Синхронизация и её методы (`UpdateFromSync`, `ListExternalIDs`) удалены.
6. Compose: dev — без сервиса и профиля `keycloak`. Prod — без `keycloak`, `keycloak-db`, их томов и переменных;
   `app` на сети хоста (`network_mode: host`, без `networks` и `ports`), `server.host: 127.0.0.1` в
   `configs/config.prod.yaml`; MongoDB и Redis публикуются на `127.0.0.1`; URI с `directConnection=true`, чтобы
   существующий член replica set `mongodb:27017` не мешал. Dev и prod стеки на одной машине одновременно не
   поднимаются (порты 27017/6379). Доступ снаружи хоста — открытый вопрос владельца.
7. Документы: AGENTS.md (раздел «Token Auth Mode» → «Authentication»: без `auth.mode`, правило «без mock/static
   запасных путей, loopback» сохраняется), README, docs/*, шаблоны. В `docs/ARCHITECTURE.md` — чего не хватает для
   нескольких пользователей.

## Technical Details

Чего не хватает для «несколько токенов → разные пользователи» (в документацию, не строить):
- у токена в конфиге нет личности: нужно `tokens[].user {username, email}` и `ExternalUserID` из токена вместо
  константы `owner`; проверка уникальности `username/email` между токенами;
- `userResolver` при конфликте имени находит существующего пользователя по `username`: токен с чужим `username`
  молча входит как этот пользователь — для нескольких личностей это надо запретить;
- `AcceptInviteUseCase` не подключён к HTTP и членства не даёт; добавить человека можно только существующего
  пользователя через форму приглашения в workspace, а выдать ему токен — только правкой конфига и перезапуском;
- loopback: команда не на одной машине — нужен решённый сетевой доступ с TLS.

Что перестаёт проверяться (в отчёт и `tests/README.md`): OAuth-вход и обмен кода, проверка JWT Keycloak (JWKS,
издатель, аудитория), группы Keycloak при создании workspace и принятии приглашения, синхронизация пользователей,
импорт realm, `/auth/login` и `/auth/refresh` в e2e. Всё удаляется вместе с кодом.

Существующая база с данными времён Keycloak (если есть в проде): пользователь владельца хранит `sub` Keycloak в
`keycloak_id`, а токен даёт `owner`. Шаг миграции для `DEPLOYMENT.md`:
`db.users.updateOne({username: "<owner>"}, {$set: {keycloak_id: "owner"}})`.

## Implementation Steps

### Task 1: Workspace без групп Keycloak

- [x] убрать `keycloakGroupID` из агрегата, `Reconstruct`, события, документа, `FindByKeycloakGroup` и индекс
- [x] убрать `KeycloakClient`, `NoOpKeycloakClient`, `ListUserWorkspacesUseCase` и проводку
- [x] обновить вызовы `NewWorkspace` во всех тестах
- [x] тесты: создатель — владелец; приглашение отмечается использованным; старый документ с `keycloak_group_id` читается
- [x] `go build`, `go vet -tags=integration,e2e`, `go test ./...`

### Task 2: Воркер и пользователи без синхронизации

**Files:** `internal/worker/runner.go`; Delete: `user_sync.go`, `user_sync_test.go`, `runner_internal_test.go`;
`internal/domain/user/user.go`, `internal/application/user/commands.go`,
`internal/infrastructure/repository/mongodb/user_repository.go` (+ тесты); `internal/application/workspace/queries.go`

- [ ] убрать `UserSyncWorker` из воркера (тесты воркера были только про синхронизацию)
- [ ] удалить `UpdateFromSync`, `ListExternalIDs` и их тесты; Go-поле `KeycloakID` → `ExternalID`, тег прежний
- [ ] удалить `ListUserWorkspacesQuery`
- [ ] `go build`, `go vet -tags=integration,e2e`, `go test ./...`

### Task 3: Вход только токеном в HTTP и проводке

**Files:** `cmd/api/container.go`, `routes.go` (+ тесты); `internal/middleware/auth.go`, удалить
`keycloak_adapter.go`; `internal/handler/http/{auth_handler,auth_middleware,auth_cookies,template_handler}.go`
(+ тесты); шаблоны `web/templates/auth/{login,callback}.html`; Delete: `internal/infrastructure/keycloak/`,
`internal/service/auth_service.go`, `internal/infrastructure/auth/`, `tests/integration/keycloak_*_test.go`,
`tests/testutil/keycloak.go`; `tests/e2e/setup_test.go`, `tests/e2e/auth_test.go`;
`docs/api/{openapi.yaml,postman_collection.json,README.md}`

- [ ] `setupTokenValidator` → только `OwnerTokenValidator`; удалить `createAuthService`, OAuth-адаптер
- [ ] удалить mock-сессию, `setMockUserContext`, `RequireValidator`, `StaticTokenValidator`, `MockAuthService`, `DisabledAuthService`
- [ ] `TemplateHandler`: только форма токена; удалить `/auth/callback`, `GetRedirectURI`, шаблон callback
- [ ] API: удалить `login/refresh/logout` и их `SkipPaths`/`AllowExpiredForPaths`; оставить `me`; OpenAPI
- [ ] удалить пакет keycloak, `AuthService`, token store, интеграционные тесты Keycloak и контейнер в testutil
- [ ] e2e: `setup_test.go` без `MockAuthService`; `auth_test.go` — оставить `me` и отказы без токена, вход/обновление удалить
- [ ] тесты: страница входа (верный/неверный токен, просроченный state), middleware без валидатора отклоняет, `/auth/me`
- [ ] `go build`, `go vet -tags=integration,e2e`, `go test ./...`, `go test -tags=e2e ./tests/e2e/` (без frontend)

### Task 4: Конфиг без режимов входа

**Files:** `internal/config/config.go` (+ тест), `cmd/api/main.go` (+ `main_test.go`), `cmd/api/container.go`
(+ тест), `configs/config.yaml`, `config.dev.yaml`, `config.prod.yaml`; Delete: `configs/config.token.yaml`;
`scripts/dev-full-stack.sh`, `Makefile`

- [ ] удалить `KeycloakConfig`, `AuthMode`, `JWTSecret`, TTL токенов, `IsTokenMode`; `DefaultHost` = `127.0.0.1`
- [ ] `ValidateAuth()` с проверками бывшего режима токена; вызов из `cmd/api` до сборки контейнера
- [ ] удалить `IsProduction`, `ErrMockModeInProd`; mock-проверка доступа — только при `app.mode: mock`; `getEnvironment`
- [ ] `config.yaml` = бывший `config.token.yaml`; скрипт и `make dev`/`dev-local`/`dev-lite` на нём
- [ ] тесты: `ValidateAuth` (нет токена, владельца, loopback, `session_ttl`); `Validate` их не требует; старый YAML с `keycloak:` и `auth.mode` загружается; mock-проверка доступа вне mock-режима отклоняется
- [ ] `go build`, `go vet -tags=integration,e2e`, `go test ./...`

### Task 5: Compose, скрипты, go.mod

**Files:** `docker-compose.yml`, `docker-compose.prod.yml`, `internal/config/docker_compose_prod_test.go`,
`.env.example`, `.golangci.yml`; Delete: `scripts/setup-keycloak.sh`, `configs/keycloak/`, `configs/keycloak-prod/`

- [ ] dev compose без сервиса и профиля `keycloak`
- [ ] prod compose: без Keycloak, `app` с `network_mode: host`, без `networks`/`ports`; MongoDB/Redis на `127.0.0.1`; URI `directConnection=true`; обязательная переменная — `AUTH_OWNER_TOKEN_SHA256`
- [ ] тест prod compose: нет Keycloak, `network_mode: host`, loopback-порты, обязательные переменные
- [ ] `go mod tidy`, записать убранные зависимости
- [ ] `docker compose -f docker-compose.prod.yml config`; `go build`, `go vet -tags=integration,e2e`, `go test ./...`

### Task 6: Браузерные e2e на токен

**Files:** `tests/e2e/frontend/frontend_test.go`, `smoke_regression_test.go`, `README.md`; `tests/README.md`

- [ ] `loginAsTestUser`: форма токена, токен из `E2E_TOKEN`, без него — пропуск с причиной
- [ ] `tests/README.md`: что перестало проверяться
- [ ] `go vet -tags=e2e ./tests/e2e/frontend/`; прогон против живого `make dev` с `E2E_TOKEN`, если Playwright установлен

### Task 7: Документация

- [ ] AGENTS.md, README.md, docs/ARCHITECTURE.md, DEPLOYMENT.md (шаг миграции владельца), DEVELOPMENT.md,
  development/setup.md, USER_GUIDE.md, min-infra-review.md, `web/templates/home.html`, `tests/load/tag-system/README.md`
- [ ] раздел «несколько пользователей: чего не хватает» в `docs/ARCHITECTURE.md`

### Task 8: Verify acceptance criteria

- [ ] `command grep -rni keycloak` вне `docs/plans/completed`: только тег `bson:"keycloak_id"`, имя индекса
  `idx_users_keycloak_unique`, фикстура `keycloak_group_id` в тесте старого документа, упоминания в этом плане
- [ ] `make lint` (новые замечания относительно `fix/write-contract`), `go test ./...`, интеграционные и e2e API
- [ ] живой `make dev-local`: вход формой токена, лента, действие с ответом `{chat_id, version, replayed}`
- [ ] живой prod compose с токеном: `/health`, запрос с Bearer
- [ ] перенести план в `docs/plans/completed/`

## Post-Completion

- Существующая база: workspace с `keycloak_group_id` и разреженный индекс `idx_workspaces_keycloak_unique`
  безвредны (новые документы поле не пишут); удалить вручную по желанию. Пользователь владельца — шаг миграции из
  Technical Details.
- Сетевой доступ к prod-стеку снаружи хоста — открытый вопрос владельца.
