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
