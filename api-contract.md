# API-контракт SkillSwap (ЛР3)

Общие правила (identity, request ID, бюджеты, shutdown) — в README, раздел D.
Здесь — маршруты участника A. Остальные участники дополняют свои разделы.

## A. Подтверждение договорённости и чтение её состояния

Договорённость — это отклик (`Response`) на предложение (`Offer`). Она создаётся
фикстурой при запуске: `response-1` от `learner` на `offer-1` автора `author`,
статус `pending`.

### Общий вход

| Что | Правило |
| --- | --- |
| Actor | Только `X-Demo-User` (middleware D). JSON и query не могут назначить actor |
| `{id}` в пути | `^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`, иначе `400 invalid_id` |
| Query | Запрещены любые параметры (включая пустой `?`) → `400 query_not_allowed` |
| Тело | Запрещено на обоих маршрутах; читается не больше 1 байта → `400 body_not_allowed` |

Неверный вход отклоняется до вызова use case и не меняет состояние.

### `PUT /v1/engagements/{id}/acceptance`

Автор предложения подтверждает отклик. PUT выбран потому, что команда
устанавливает известное итоговое состояние `accepted`; идемпотентность
обеспечивает use case, а не метод.

Порядок: чтение договорённости → ранний отказ не-автору (без сети) →
справочник предложений (зависимость C) → атомарный переход
`pending → accepted` под блокировкой store (`AcceptContext`, D).

```bash
curl -i -X PUT -H 'X-Demo-User: author' http://127.0.0.1:8080/v1/engagements/response-1/acceptance
```

`200 OK`:

```json
{"id":"response-1","offer_id":"offer-1","author_id":"author","learner_id":"learner","status":"accepted","changed":true}
```

Повтор при неизменных условиях — no-op: тот же ответ с `"changed":false`,
новой записи нет. Если условия изменились (например, предложение закрыто),
повтор получает предметный отказ по таблице ниже.

### `GET /v1/engagements/{id}`

Читает тот же store, что меняет команда. Ничего не пишет и зависимость не
вызывает. Доступно участникам: автору предложения и автору отклика.

```bash
curl -i -H 'X-Demo-User: learner' http://127.0.0.1:8080/v1/engagements/response-1
```

`200 OK`:

```json
{"id":"response-1","offer_id":"offer-1","author_id":"author","learner_id":"learner","status":"accepted"}
```

`status`: `pending` | `accepted` | `rejected`. Отсутствие договорённости — это
предметная ошибка `404 engagement_not_found`, а не начальное состояние:
договорённости не возникают «пустыми», их создаёт фикстура.

### DTO

Публикуются только `id`, `offer_id`, `author_id`, `learner_id`, `status`
(и `changed` у команды). Навыки, формат и статус предложения не публикуются.

### Ошибки

Форма: `{"code":"engagement_not_found","message":"Договорённость не найдена"}`.
Причины сопоставляются через `errors.Is`, поэтому обёрнутые ошибки сохраняют
статус и code. Внутренний `err.Error()` пишется только в журнал вместе с
`request_id`.

| Причина | Статус | code |
| --- | --- | --- |
| Нет / неверный `X-Demo-User` | 400 | `invalid_demo_user` |
| Неверный `{id}` | 400 | `invalid_id` |
| Есть query | 400 | `query_not_allowed` |
| Непустое тело | 400 | `body_not_allowed` |
| Отклик не найден (`domain.ErrNotFound`) | 404 | `engagement_not_found` |
| Справочник не знает предложение (`ErrOfferMissing`, режим `missing`) | 404 | `offer_not_found` |
| Подтверждает не автор предложения (`ErrNotOfferAuthor`) | 403 | `not_offer_author` |
| Читает не участник (`ErrNotParticipant`) | 403 | `not_participant` |
| Справочник: предложение неактивно (`ErrOfferUnavailable`) | 409 | `offer_unavailable` |
| Переход запрещён: отклик `rejected`, предложение `closed` (`ErrInvalidTransition`) | 409 | `invalid_transition` |
| Справочник: 5xx, сеть, неверный/большой ответ, чужой ID (`ErrDirectoryFailed`, режимы `error`, `large`) | 502 | `dependency_failed` |
| Истёк бюджет запроса (`context.DeadlineExceeded`, режим `slow`) | 504 | `dependency_timeout` |
| Клиент отменил запрос (`context.Canceled`) | 503 | `request_canceled` (доставка не гарантируется) |
| Прочее | 500 | `internal_error` |

Неизвестный путь или метод получает стандартный ответ `ServeMux` 404/405 —
это исключение из JSON-формата.

### Стык с C: `agreement.OfferDirectory`

```go
type OfferDirectory interface {
	OfferInfo(ctx context.Context, offerID string) (agreement.OfferInfo, error)
}
type OfferInfo struct { ID string; Active bool }
```

Адаптер C обязан: передавать `ctx` в запрос; на 404 возвращать ошибку,
оборачивающую `domain.ErrNotFound`; на 5xx, сетевую ошибку, превышение лимита
тела или невалидный JSON — оборачивающую `agreement.ErrDirectoryFailed`;
на истечение ctx — оборачивающую `ctx.Err()`. `OfferInfo.ID` сверяет и A.
Пока клиент C не подключён, в `cmd/api` используется
`agreement.LocalOfferDirectory` поверх общего store.
