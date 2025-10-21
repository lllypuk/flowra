# UseCase Layer - Priority Tasks

**Последнее обновление:** 2025-10-22
**Прогресс:** 82% завершено

---

## 🔴 КРИТИЧНО - Выполнить немедленно

### Task 09: Chat UseCases Testing
**Статус:** 🔴 NOT STARTED
**Время:** 3-4 часа
**Блокирует:** Переход к infrastructure layer

**Проблема:**
- Chat UseCases имеют **0% test coverage**
- 12 реализованных UseCases без единого теста
- Критический риск для проекта

**Действие:**
```bash
# 1. Прочитать план
cat docs/tasks/04-impl-usecase/09-chat-tests.md

# 2. Создать тесты для всех 12 UseCases
# 3. Достичь coverage >85%
# 4. Убедиться все тесты проходят
```

**Детали:** [09-chat-tests.md](./09-chat-tests.md)

**Метрика успеха:**
- ✅ 60+ unit тестов
- ✅ Coverage: 0% → >85%
- ✅ Application layer coverage: 64.7% → ~75%

---

## 🟡 ВЫСОКИЙ ПРИОРИТЕТ - Выполнить после Task 09

### Task 10: Chat Query UseCases Implementation
**Статус:** 🟡 NOT STARTED
**Время:** 1-2 часа
**Блокирует:** Полную функциональность Chat

**Проблема:**
- Отсутствуют Query UseCases
- Невозможно получить данные для UI
- Phase 2 неполная

**Действие:**
```bash
# 1. Прочитать план
cat docs/tasks/04-impl-usecase/10-chat-queries.md

# 2. Реализовать 3 Query UseCases:
#    - GetChatUseCase
#    - ListChatsUseCase
#    - ListParticipantsUseCase

# 3. Написать 15 тестов
```

**Детали:** [10-chat-queries.md](./10-chat-queries.md)

**Метрика успеха:**
- ✅ 3 Query UseCases реализованы
- ✅ 15 unit тестов
- ✅ Phase 2 полностью завершена

---

## 📝 СРЕДНИЙ ПРИОРИТЕТ - Рекомендуется

### Task 11: Documentation Update
**Статус:** 📝 OPTIONAL
**Время:** 1 час

**Обновить:**
- [ ] README.md
- [ ] ARCHITECTURE_DIAGRAM.md
- [ ] Создать API_EXAMPLES.md

---

## 🧪 НИЗКИЙ ПРИОРИТЕТ - Опционально

### Task 12: E2E Tests
**Статус:** 🧪 OPTIONAL
**Время:** 2-3 часа

**Создать:**
- [ ] Complete Task workflow test
- [ ] Messaging workflow test
- [ ] Workspace invitation workflow test

---

## ⏸️ ОТЛОЖЕНО - Сделать в infrastructure phase

### Task 13: Notification Event Handlers
**Статус:** ⏸️ POSTPONED
**Время:** 2 часа
**Причина:** Требует Event Bus implementation

### Task 14: CI/CD Setup
**Статус:** ⏸️ POSTPONED
**Время:** 1-2 часа
**Причина:** Можно сделать параллельно с infrastructure

---

## 🎯 Быстрый старт

### Сегодня (обязательно):
```bash
# 1. Chat Tests (3.5 часа)
cd internal/application/chat
# Следовать плану в 09-chat-tests.md
go test -v -coverprofile=coverage.out ./...

# 2. Query UseCases (2 часа)
# Следовать плану в 10-chat-queries.md
go test -v -run Query ./...
```

### Проверка прогресса:
```bash
# Общий coverage
go test -coverprofile=/tmp/coverage.out ./internal/application/...
go tool cover -func=/tmp/coverage.out | grep total

# Должно быть: >75% (сейчас 64.7%)
```

### После завершения:
```bash
# Обновить трекер
vim docs/tasks/04-impl-usecase/PROGRESS_TRACKER.md
# Отметить Phase 2 как завершённую

# Готово к переходу на infrastructure layer!
```

---

## 📊 Метрики

### Текущее состояние:
```
✅ Завершено:     5/8 фаз (62.5%)
🟡 В процессе:    3/8 фаз (37.5%)
❌ Не начато:     0/8 фаз

Overall: 82% complete

Критические задачи:
🔴 Task 09: Chat Tests        - NOT STARTED
🟡 Task 10: Query UseCases    - NOT STARTED
```

### Целевое состояние (после Task 09+10):
```
✅ Завершено:     7/8 фаз (87.5%)
🟡 В процессе:    1/8 фаз (12.5%)

Overall: ~95% complete

Application coverage: 64.7% → ~82%
Chat coverage: 0% → >85%
```

---

## 🚀 План на неделю

### День 1 (СЕГОДНЯ):
- [x] Проанализировать состояние
- [x] Создать планы
- [ ] **Task 09: Chat Tests (3.5 часа)** ← КРИТИЧНО

### День 2:
- [ ] **Task 10: Query UseCases (2 часа)** ← ВЫСОКИЙ ПРИОРИТЕТ
- [ ] Task 11: Documentation (1 час) - опционально

### День 3+:
- [ ] Task 12: E2E Tests - опционально
- [ ] Переход к infrastructure layer

---

## 📞 Помощь

**Если застрял:**

1. **Изучи примеры:**
   - `internal/application/message/*_test.go` - примеры тестов
   - `internal/application/user/` - примеры Query UseCases
   - `tests/mocks/` - готовые mocks

2. **Проверь документацию:**
   - [09-chat-tests.md](./09-chat-tests.md) - детальный план тестов
   - [10-chat-queries.md](./10-chat-queries.md) - детальный план Query
   - [COMPLETION_PLAN.md](./COMPLETION_PLAN.md) - общий план

3. **Используй готовую инфраструктуру:**
   - Mocks уже созданы в `tests/mocks/`
   - Fixtures готовы в `tests/fixtures/`
   - Test utilities в `tests/testutil/`

---

## ✅ Definition of Done

UseCase layer считается **полностью завершённым** когда:

- [x] Phase 1: Architecture - 100%
- [ ] **Phase 2: Chat UseCases - 100%** ← В РАБОТЕ
  - [ ] All Command UseCases tested
  - [ ] All Query UseCases implemented
  - [ ] Coverage >85%
- [x] Phase 3: Message UseCases - 100%
- [x] Phase 4: User UseCases - 100%
- [x] Phase 5: Workspace UseCases - 100%
- [x] Phase 6: Notification UseCases - 100% (UseCases only)
- [x] Phase 8: Tag Integration - 100%
- [ ] Phase 7: Integration Testing - 80%+ (базовое тестирование)

**Минимальная граница:** Task 09 + Task 10 = **готово к infrastructure layer**

---

**Сфокусируйся на Task 09 и Task 10 - это единственное, что блокирует прогресс!** 🎯
