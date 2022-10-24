# Микросервис для работы с балансом пользователей

## Запуск сервиса

```shell
docker-compose build
docker-compose up
```

## SQL для создания базы данных
```sql
CREATE TABLE "deposit" (
  "id" SERIAL PRIMARY KEY,
  "user_id" int NOT NULL,
  "amount" int NOT NULL,
  "deposit_at" timestamptz DEFAULT (now())
);

CREATE TABLE "payment" (
  "id" SERIAL PRIMARY KEY,
  "user_id" int NOT NULL,
  "order_id" int NOT NULL,
  "service_id" int NOT NULL,
  "amount" int NOT NULL,
  "reserved_at" timestamptz DEFAULT (now()),
  "confirmed_at" timestamptz,
  "canceled_at" timestamptz
);

CREATE TABLE "money_send" (
  "id" SERIAL PRIMARY KEY,
  "from" int NOT NULL,
  "to" int NOT NULL,
  "amount" int NOT NULL,
  "sent_at" timestamptz DEFAULT (now())
);

CREATE INDEX ON "deposit" ("user_id");

CREATE UNIQUE INDEX ON "payment" ("user_id", "order_id", "service_id");

CREATE INDEX ON "payment" ("user_id");

CREATE INDEX ON "money_send" ("from");

CREATE INDEX ON "money_send" ("to");
```

## Примеры запросов

[![Run in Postman](https://run.pstmn.io/button.svg)](https://app.getpostman.com/run-collection/e9d1c4df814f61af69f6?action=collection%2Fimport)

## Реализованные задачи
- зачисление средств
- списание средств
  - резервирование средств
  - подтверждение транзакции
  - отмена транзакции
- перевод средств от пользователя к пользователю
- получения баланса пользователя
- получение детализации баланса пользователя

## TODO:
- сводный отчет для бухгалтерии
- документация в Swagger
- покрытие кода тестами