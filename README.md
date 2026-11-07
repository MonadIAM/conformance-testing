# @monadiam/conformance-testing

Локальный запуск OpenID Conformance Suite против MonadIAM. Go CLI, Docker Compose, nginx, MongoDB.

----

<details>
<summary><strong>Запуск</strong></summary>

Требования: Go 1.27+, Docker Compose v2, Make, curl.

Identity запущен отдельно на HTTP-порту `4002` с `JWT_ISSUER=https://localhost:4002`.

```sh
make pull
make up
make test
```

Конфигурация: `configs/local.example.json`. Подготовленный профиль: Config OP.

TLS завершается на `iam-loopback` внутри сети suite. Прокси обращается к Identity по HTTP. Браузерный вход и mTLS не настроены.

</details>

----

<details>
<summary><strong>Команды</strong></summary>

| Команда             | Назначение                                       |
| :------------------ | :----------------------------------------------- |
| `make build`        | Сборка `bin/conformance`                         |
| `make check`        | Compose, shell-синтаксис, gofmt, go vet          |
| `make fmt`          | Форматирование Go                                |
| `make pull`         | Загрузка образов                                 |
| `make up`           | Запуск стенда                                    |
| `make stop`         | Остановка контейнеров                            |
| `make down`         | Удаление контейнеров и сети с сохранением данных |
| `make status`       | Состояние контейнеров                            |
| `make logs`         | Логи Java suite и прокси                         |
| `make smoke`        | Проверка HTTPS suite                             |
| `make check-iam`    | Получение discovery из контейнера suite          |
| `make test`         | Запуск Config OP                                 |
| `make conformance`  | Запуск плана с `PLAN`, `CONFIG`, `ARGS`          |

```sh
make conformance PLAN=oidcc-config-certification-test-plan CONFIG=configs/local.json
./bin/conformance -help
```

Модули выполняются последовательно. Таймаут CLI не останавливает тест в suite.

Коды завершения CLI:

- `0` — все выбранные модули PASSED.
- `1` — другой результат теста.
- `2` — ошибка выполнения или таймаут.

</details>

----
