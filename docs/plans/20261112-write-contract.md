# Контракт записи: принятое изменение задачи не теряется

## Overview

Задача лида T-0013, шаг 2 рекомендации из `docs/min-infra-review.md` (раздел 5, направление согласовано с Codex).
Сейчас `POST .../actions/{status,priority,assignee,due-date,close,reopen}` пишет в тред системное сообщение с тегом
и отвечает 200, а тег исполняется в отвязанной горутине после ответа: при 50 агентах на одну задачу теряется 40%
принятых изменений, вызывающий об этом не узнаёт. `actions/rename` чат не переименовывает.

Что должно получиться:
- ответ на действие — определённый итог: применено (с версией агрегата), отказ (клиент видит код) или неизвестный исход;
- повтор доменной команды с перезагрузкой агрегата при конфликте версий, ограниченный;
- ключ команды (`Idempotency-Key`): повтор после неизвестного исхода не задваивает действие;
- `rename` переименовывает чат;
- путь через теги в тексте сообщения остаётся отложенным, это названо в документации; агенты пишут через действия.

## Context (from discovery)

- `internal/service/action_service.go` — действия как сообщения `#status X` через `SendMessageUseCase`;
  `syncTaskProjection` вызывается до того, как тег применён; ошибка синхронизации проекции даёт 500.
- `internal/application/message/send_message.go:185` — `processTagsDetached`, таймаут 5 с.
- `internal/domain/tag/executor.go` — свой повтор: 5 раз, 100 мс – 2 с.
- `internal/application/chat/*.go` — use case: `Load` → доменный метод → `Save`; повторно загружают агрегат при
  каждом вызове, так что повтор вызова и есть повтор с перезагрузкой.
- `internal/infrastructure/repository/mongodb/chat_repository.go` — `Save`: события в транзакции, затем
  `chats_read_model` без проверки версии, затем outbox вне транзакции.
- `internal/infrastructure/eventstore/mongodb_store.go` — `WithTransaction`, проверка версии, уникальный индекс
  `(aggregate_id, version)`.
- `internal/infrastructure/projector/chat_to_task_read_model_projector.go` — `RebuildOne` пишет `tasks_read_model`
  без проверки версии; его зовут и шина, и `ActionService`.
- Фронтенд вызывает действия с `hx-swap="none"`: тело ответа ему безразлично.

## Development Approach

- testing approach: Regular (код, затем тесты в той же задаче).
- каждая задача заканчивается тестами; `go test ./...` зелёный перед следующей.
- ограничения: ветка `fix/write-contract` от `analysis/min-infra`, без пуша; код Keycloak не трогаем и новый код
  на него не опираем; стек лида на 8080/27017/6379 не трогаем, прогоны на своих портах.

## Solution Overview

Действие исполняется синхронно, в запросе, через `chatapp.CommandRunner`:

1. Блокировка на чат внутри процесса (ключ — `chat_id`), ожидание ограничено. Процесс один (решение владельца
   12.11), блокировка снимает конфликты версий между запросами одного процесса; проверка версии в event store
   остаётся и страхует всё остальное (тег-путь, другой процесс).
2. Под блокировкой: если задан ключ команды и событие с ним уже есть — вернуть сохранённую версию (`replayed`),
   при другом отпечатке запроса — отказ 422.
3. Вызов use case; при `ErrConcurrentModification` — повтор с перезагрузкой, до 5 попыток с джиттером.
   Исчерпано — отказ 409 «не применено».
4. Ключ и отпечаток пишутся в последний документ события той же транзакцией (`command_id`, `command_fp`),
   уникальный частичный индекс `(aggregate_id, command_id)`. Гонка двух процессов с одним ключом даёт ошибку
   дубликата, она превращается в чтение сохранённого итога.
5. Ошибка фазы фиксации транзакции (колбэк `WithTransaction` завершился, а фиксация — нет) — `ErrOutcomeUnknown`,
   503 `OUTCOME_UNKNOWN`: повторить с тем же ключом. Ошибки до фиксации — не применено.

После применения: синхронизация `tasks_read_model` и человекочитаемое сообщение в тред — лучшее усилие, их
отказ пишется в журнал и не меняет ответ «применено». Проекции защищены версией: запись `chats_read_model` и
`tasks_read_model` с версией не выше уже записанной пропускается, иначе параллельные пересборки затирают новое
старым.

Почему не очередь на агрегат вместо блокировки: та же сериализация, но больше кода (горутина-исполнитель,
остановка); блокировка с ограниченным ожиданием даёт тот же результат для одного процесса.

Тег-путь (`#status Done` в тексте сообщения) остаётся после ответа: сообщение сохранено — это и есть итог
запроса. Исполнитель тегов переходит на тот же `CommandRunner` без ключа (блокировка и ограниченный повтор
вместо своего), то есть теряет реже, но гарантии не даёт; так и записываем в документации.

## Technical Details

Ответ действий (`/chats/:id/actions/*`, `/tasks/:task_id/actions/*`):

| Итог | HTTP | Тело |
|---|---|---|
| применено | 200 | `{"success":true,"data":{"chat_id","version","replayed"}}`, заголовок `Hx-Trigger` как раньше |
| повтор ключа | 200 | то же, `replayed: true` |
| доменный отказ | 400/422 | `INVALID_INPUT`, `INVALID_STATE`, … |
| конфликт не разрешился | 409 | `CONCURRENT_MODIFICATION` — не применено, можно повторить |
| ключ с другим запросом | 422 | `IDEMPOTENCY_KEY_REUSED` |
| неизвестный исход | 503 | `OUTCOME_UNKNOWN` — повторить с тем же ключом |

- Те же итоги и ключ — у `PUT /tasks/:task_id/{status,assignee,priority,due-date}`: он идёт через тот же
  `ActionService`. Ярлыки «нет изменений» по `tasks_read_model` в обработчиках задач убираются: проекция может
  отставать, повтор того же значения домен и так делает пустым.
- Ключ: заголовок `Idempotency-Key`, до 200 символов, область — чат и автор (в `command_id` пишется
  `<actor>:<key>`), чтобы одинаковые ключи разных пользователей не выдавали чужой итог. Без ключа контракт тот же, кроме того, что
  неизвестный исход клиенту придётся проверять чтением.
- Отпечаток: `<действие>:<нормализованное значение>`.
- Передача ключа до event store — значением контекста (`appcore.WithCommand`), чтобы не менять сигнатуры
  `CommandRepository.Save` и `EventStore.SaveEvents` во всех реализациях. Другие реализации ключ игнорируют.
- `Close`/`Reopen` в домене возвращают `errors.New` → 500; оборачиваем в `errs.ErrInvalidState` → 422.
- Доступ: прежний путь через `SendMessageUseCase` отказывал не участнику чата; прямой вызов use case этой
  проверки не делает, она переносится в `ActionService`.
- Контекст: команда исполняется на отвязанном от отмены запроса контексте с одним сроком 20 с (ниже
  `WriteTimeout` 30 с): обрыв соединения клиента не делает исход неизвестным посреди фиксации.
- Ключ, по которому нечего было записывать (значение уже такое), не сохраняется: повтор после чужого изменения
  применит значение снова. Принимаем: для полей «установить значение» это и есть ожидаемое.

### Правки по ревью плана (plan-review)

Приняты: `Close`/`Reopen` шли мимо `ChatRepo` (`loadAggregate`/`saveAggregate` берут текущую версию вместо
загруженной, не обновляют `chats_read_model`, конфликт — `ErrConcurrentUpdate` → 500) — переводятся на
`CommandRepository`; проверка участника; ярлыки «нет изменений» в обработчиках задач и «status update queued»;
отвязанный контекст; при повторе по ключу и при неизвестном исходе проекции пересобираются, иначе
`chats_read_model` может остаться без зафиксированного события; защита проекций — запись при сохранённой
версии `<= v` (а не `< v`), чтобы починка той же версией работала; ключ с автором; переименование
пересобирает `tasks_read_model`. Известный предел: удаление документа `tasks_read_model` не оставляет следа,
поздняя старая пересборка вернёт документ — так было и раньше.

## Implementation Steps

### Task 1: Ключ команды и неизвестный исход в event store

**Files:**
- Modify: `internal/application/appcore/eventstore.go`
- Modify: `internal/infrastructure/eventstore/mongodb_store.go`, `serializer.go` (`EventDocument`)
- Modify: `internal/infrastructure/mongodb/indexes.go`
- Modify: `internal/domain/errs/*.go`
- Create/Modify: тесты event store (testcontainers)

- [ ] `appcore.WithCommand`/`CommandFrom`, `ErrDuplicateCommand`, `errs.ErrOutcomeUnknown`, `errs.ErrIdempotencyKeyReused`
- [ ] `EventDocument.CommandID`, `CommandFP` (omitempty), ставятся на последний документ пачки
- [ ] индекс `idx_events_aggregate_command_unique`, частичный по `command_id`
- [ ] `SaveEvents`: дубликат по этому индексу → `ErrDuplicateCommand`; ошибка после завершения колбэка → `ErrOutcomeUnknown`
- [ ] `FindCommand(ctx, aggregateID, commandID) (version int, fp string, found bool, err error)`
- [ ] тесты: запись с ключом и поиск; повтор ключа → `ErrDuplicateCommand`; без ключа как раньше

### Task 2: `CommandRunner`

**Files:**
- Create: `internal/application/chat/command_runner.go`, `command_runner_test.go`

- [ ] блокировка на чат с ограниченным ожиданием и освобождением записи после последнего пользователя
- [ ] поиск ключа под блокировкой, отпечаток, повтор с перезагрузкой до 5 попыток, джиттер
- [ ] `ErrDuplicateCommand` → чтение сохранённого итога
- [ ] тесты: применено; конфликт разрешился повтором; исчерпано → `ErrConcurrentModification`; повтор ключа;
  другой отпечаток; неизвестный исход не повторяется; параллельные вызовы на один чат сериализуются

### Task 3: Проекции с проверкой версии

**Files:**
- Modify: `internal/infrastructure/repository/mongodb/chat_repository.go`
- Modify: `internal/infrastructure/projector/chat_projector.go`, `chat_to_task_read_model_projector.go`

- [ ] `chats_read_model` хранит `version`; запись при сохранённой `version <= v` или без поля; дубликат ключа при upsert — пропуск
- [ ] то же для `tasks_read_model` (поле `version` там уже есть)
- [ ] тесты: старая версия не затирает новую; документ без версии обновляется

### Task 4: `ActionService` исполняет команды синхронно

**Files:**
- Modify: `internal/service/action_service.go`, `action_service_test.go`
- Modify: `internal/application/appcore/action_result.go`
- Modify: `internal/domain/chat/chat.go` (ошибки `Close`/`Reopen`)
- Modify: `cmd/api/container.go`

- [ ] статус, приоритет, исполнитель, срок, закрыть, открыть, переименовать — через `CommandRunner` и use case;
  системное сообщение с тегом больше не пишется
- [ ] `ActionResult` несёт `ChatID`, `Version`, `Replayed`; ключ и отпечаток приходят параметром
- [ ] сбой синхронизации проекции и сообщения в тред не превращает «применено» в ошибку
- [ ] `CloseChatUseCase`, `ReopenChatUseCase` — через `CommandRepository`
- [ ] проверка участника чата; отвязанный контекст со сроком
- [ ] после повтора по ключу — пересборка обеих проекций
- [ ] тесты: каждое действие применяет событие; `rename` меняет имя и проекцию задачи; не участник — отказ;
  сбой проекции не меняет «применено»; ошибки use case доходят до вызывающего

### Task 5: HTTP-ответ с итогом

**Files:**
- Modify: `internal/handler/http/chat_action_handler.go`, `task_action_handler.go`, `task_handler.go` и тесты
- Modify: `internal/infrastructure/httpserver/response.go`

- [ ] читать `Idempotency-Key`, проверять длину; то же в `PUT /tasks/...`
- [ ] убрать ярлыки «нет изменений» и «status update queued» в `task_action_handler.go`, `task_handler.go`
- [ ] 200 с телом итога; `Hx-Trigger` сохранить
- [ ] `mapError`: `ErrOutcomeUnknown` → 503, `ErrIdempotencyKeyReused` → 422
- [ ] тесты обработчиков и отображения ошибок

### Task 6: Исполнитель тегов через `CommandRunner`

**Files:**
- Modify: `internal/domain/tag/executor.go`, тесты; `cmd/api/container.go`

- [ ] заменить свой повтор на `CommandRunner` без ключа
- [ ] тесты исполнителя проходят

### Task 7: agentbench и приёмка

**Files:**
- Modify: `tests/load/agentbench/main.go`, `run.sh`

- [ ] `contend`: ключ на каждый запрос, запись принятых значений и кодов ответа
- [ ] сверка: каждое принятое значение срока ровно одним событием, событий без принятого запроса нет (кроме
  неизвестных исходов), итог в `chats_read_model` и `tasks_read_model` равен последнему событию
- [ ] `BENCH_API_BIN` для прогона «до» на бинаре базовой ветки
- [ ] прогоны: 1, 2, 10, 50 агентов × 3 до и после; пик и стресс × 3 до и после

### Task 8: Verify acceptance criteria

- [ ] `make lint`, `go test ./...`, интеграционные тесты затронутых пакетов
- [ ] таблицы до/после совпадают с требованиями Overview

### Task 9: [Final] Update documentation

- [ ] `docs/api/action-endpoints.md`, `openapi.yaml`: итоги, коды, `Idempotency-Key`, тег-путь отложенный
- [ ] `docs/min-infra-review.md`: результаты приёмки, ссылка на замеры
- [ ] план — в `docs/plans/completed/`

## Post-Completion

- Удаление Keycloak — следующая задача лида.
- Outbox пишется вне транзакции событий: при сбое между ними события не дойдут до шины (WebSocket, уведомления);
  `tasks_read_model` в пути действий синхронизируется сам. Не в этой задаче.
- Остальные пути записи чата (`PUT /chats/:id`, участники) — прежние: оптимистичная проверка, 409 без повтора.
