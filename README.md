# URL Shortener

Сервис сокращения URL на Go - пет-проект для портфолио, демонстрирующий продакшн-практики бэкенда: чистую архитектуру хендлеров и хранилища, юнит-тесты с моками зависимостей, контейнеризованный деплой и CI/CD-пайплайн, доставляющий изменения на VPS.

## Демо

Сервис задеплоен и работает:

**Проверить:** [http://139.100.238.51:8082/pwnz](http://139.100.238.51:8082/pwnz) - редиректит на этот репозиторий

> `POST /url` и `DELETE /url/{alias}` требуют Basic Auth и предназначены только для владельца сервиса - эта демо-ссылка задействует только публичную ручку редиректа.

## Возможности

- Создание короткой ссылки с опциональным кастомным alias (если не указан - генерируется случайный, 6 символов)
- Редирект по alias на оригинальный URL
- Удаление ссылки по alias
- Мутирующие ручки (`POST`, `DELETE`) защищены HTTP Basic Auth; ручка редиректа публичная, как и положено для URL-шортенера
- Структурированное логирование через `slog`: человекочитаемый формат локально, JSON в проде
- Хранилище на SQLite (чистый Go-драйвер, без CGO)
- Многоэтапная сборка в Docker, деплой на VPS через пайплайн GitHub Actions

## Стек

- **Go 1.26**
- [`chi`](https://github.com/go-chi/chi) - роутер и middleware
- [`go-playground/validator`](https://github.com/go-playground/validator) - валидация запросов
- [`cleanenv`](https://github.com/ilyakaznacheev/cleanenv) - загрузка YAML-конфига
- [`modernc.org/sqlite`](https://gitlab.com/cznic/sqlite) - чистый Go-драйвер SQLite (CGO не требуется)
- `log/slog` + [`tint`](https://github.com/lmittmann/tint) - структурированные логи, красиво отформатированные локально
- [`testify`](https://github.com/stretchr/testify) + [`mockery`](https://github.com/vektra/mockery) - юнит-тесты со сгенерированными моками
- Docker, GitHub Actions, GHCR (GitHub Container Registry)

## API

Все ответы используют единый конверт:

```json
{ "status": "OK" }
```
```json
{ "status": "Error", "error": "описание того, что пошло не так" }
```

### `POST /url` - создать короткую ссылку 🔒 требует Basic Auth

**Запрос**
```json
{
  "url": "https://example.com/some/very/long/path",
  "alias": "custom-alias"
}
```
`alias` опционален - если не указан, генерируется случайный 6-символьный.

**Ответ `200`**
```json
{ "status": "OK", "alias": "custom-alias" }
```

Возможные ошибки: невалидный/отсутствующий `url` (ошибка валидации), alias уже существует, внутренняя ошибка - все возвращаются с кодом `200` и телом `status: "Error"`, согласно конверту выше.

### `DELETE /url/{alias}` - удалить короткую ссылку 🔒 требует Basic Auth

**Ответ `200`**
```json
{ "status": "OK" }
```
Возвращает тело с ошибкой, если alias не существует.

### `GET /{alias}` - редирект на оригинальный URL - публично, без авторизации

Отвечает `302 Found` с заголовком `Location`, указывающим на оригинальный URL. Возвращает тело с ошибкой, если alias не существует. Эта ручка намеренно вынесена за пределы защищённой группы роутов - по короткой ссылке должен быть способен перейти кто угодно.

## Конфигурация

Конфиг загружается из YAML-файла при старте. Путь берётся из переменной окружения `CONFIG_PATH` - процесс сразу завершается, если она не задана или файла не существует.

| Поле | YAML-ключ | Переопределение через env | Обязательно | Описание |
|---|---|---|---|---|
| Env | `env` | - | нет (по умолчанию `local`) | `local` \| `dev` \| `prod` - влияет только на формат/уровень логов |
| StoragePath | `storage_path` | - | да | Путь к файлу базы данных SQLite |
| HTTPServer.Address | `http_server.address` | - | нет (по умолчанию `:8080`) | Адрес для прослушивания |
| HTTPServer.Timeout | `http_server.timeout` | - | нет (по умолчанию `4s`) | Таймаут чтения/записи |
| HTTPServer.IdleTimeout | `http_server.idle_timeout` | - | нет (по умолчанию `60s`) | Таймаут простаивающего соединения |
| HTTPServer.User | `http_server.user` | - | да | Логин для Basic Auth |
| HTTPServer.Password | `http_server.password` | `HTTP_SERVER_PASSWORD` | да | Пароль для Basic Auth |

Есть два конфиг-файла: [`config/local.yaml`](config/local.yaml) для локальной разработки и [`config/prod.yaml`](config/prod.yaml) - для задеплоенного контейнера. Продовый пароль **не** хранится в `prod.yaml` - он подставляется во время запуска через переменную окружения `HTTP_SERVER_PASSWORD`.

## Запуск локально

```bash
export CONFIG_PATH=./config/local.yaml
go run ./cmd/url-shortener
```

Сервер стартует на `localhost:8082` (как задано в `config/local.yaml`), с красиво отформатированным логом в консоли.

## Тесты

```bash
go test ./...
```

Хендлеры тестируются изолированно, против замоканных интерфейсов хранилища (моки сгенерированы `mockery` - см. директивы `//go:generate` в каждом хендлере). Хендлер редиректа дополнительно покрыт end-to-end тестом, который поднимает настоящий `chi`-роутер и HTTP-сервер, проверяя реальное поведение `302`/`Location`, не переходя по самому редиректу.

## Docker

```bash
docker build -t url-shortener .

docker run -d \
  --name url-shortener \
  -p 8082:8082 \
  -v $(pwd)/config:/app/config:ro \
  -v $(pwd)/data:/app/data \
  -e CONFIG_PATH=/app/config/prod.yaml \
  -e HTTP_SERVER_PASSWORD=ваш-пароль \
  url-shortener
```

Образ запускается от непривилегированного пользователя с зафиксированным UID/GID (`10001`) - поэтому директория на хосте, смонтированная в `/app/data`, должна быть доступна на запись именно этому UID (см. [`Dockerfile`](Dockerfile)).

## CI/CD и деплой

Деплой ручной, по тегу - через workflow [`Deploy App`](.github/workflows/deploy.yml) (`workflow_dispatch`, запускается из вкладки Actions с git-тегом в качестве входного параметра):

1. **`build-and-push`** - чекаутит указанный тег, проверяет его существование, собирает Docker-образ через Buildx и пушит в GHCR с тегами версии и `latest`.
2. **`deploy`** - по SSH: убеждается, что нужные директории на сервере существуют и принадлежат UID контейнера, копирует актуальный `config/prod.yaml` на сервер, затем стягивает новый образ и перезапускает контейнер (`--restart unless-stopped`).

Обязательные секреты репозитория:

| Секрет | Назначение |
|---|---|
| `DEPLOY_SSH_KEY` | Приватный SSH-ключ пользователя деплоя на VPS |
| `AUTH_PASS` | Продовый пароль Basic Auth, подставляется как `HTTP_SERVER_PASSWORD` |

`GITHUB_TOKEN` предоставляется GitHub Actions автоматически и используется и для пуша в GHCR, и для авторизации `docker pull` на сервере.

> Тег нужно создавать **после** всех коммитов, которые он должен задеплоить - тег это фиксированная точка, он не "следует" за более поздними коммитами в ветке.

## Структура проекта

```
cmd/url-shortener/       - точка входа (собирает конфиг, логгер, хранилище, роутер)
internal/config/         - загрузка YAML-конфига (cleanenv)
internal/storage/        - интерфейсы и ошибки хранилища
internal/storage/sqlite/  - реализация на SQLite
internal/http-server/
  handlers/save/         - POST /url
  handlers/remove/       - DELETE /url/{alias}
  handlers/redirect/     - GET /{alias}
  middleware/mwLogger/   - middleware логирования запросов
internal/lib/
  api/                   - тестовый хелпер для проверки одного редиректа
  api/response/          - общий конверт JSON-ответа
  logger/                - хелперы для slog (форматирование ошибок, discard-логгер для тестов)
  random/                - генерация случайного alias
config/                  - local.yaml, prod.yaml
Dockerfile, .dockerignore
.github/workflows/deploy.yml
```

## Возможные улучшения

- Перенести пользователя деплоя на VPS с root на выделенного, минимально привилегированного пользователя
- Добавить health check контейнера и шаг отката в workflow деплоя, если новая версия не поднимается
- Заменить SQLite на сетевую БД, если сервису понадобится масштабироваться за пределы одного инстанса
- Добавить rate limiting на `POST /url` для защиты публичного шортенера от злоупотреблений
