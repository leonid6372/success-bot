-- +goose Up
-- +goose StatementBegin

create schema if not exists success_bot;

create or replace function success_bot.update_updated_at()
    returns trigger as $$
begin
    new.updated_at = now();
    return new;
end;
$$ language plpgsql;

create table if not exists success_bot.users
(
    id                      bigserial       primary key,

    username                varchar(32)     default ''      not null,
    first_name              varchar(64)     default ''      not null,
    last_name               varchar(64)     default ''      not null,
    language_code           varchar(2)      default 'en'    not null,
    is_premium              boolean         default false   not null,

    balance                 numeric(10, 2)  default 250000  not null,

    created_at              timestamp       default now()   not null,
    updated_at              timestamp       default now()   not null
);

create trigger update_users_updated_at
    before update on success_bot.users
    for each row
    execute function success_bot.update_updated_at();

create table if not exists success_bot.promocodes (
    id                      bigserial       primary key,
    available_count         int             default 0       not null,
    value                   varchar(64)                     not null,
    bonus_amount            numeric(10, 2)  default 0       not null,
    created_at              timestamp       default now()   not null
);

create table if not exists success_bot.instruments
(
    id                      bigserial       primary key,

    ticker                  varchar(16)     not null unique,
    name                    varchar(128)    not null
);

create table if not exists success_bot.operations
(
    id                      bigserial       primary key,

    user_id                 bigint                          not null,
    instrument_id           bigint                          not null,
    type                    varchar(16)                     not null, -- e.g., 'buy', 'sold', 'promocode'

    count                   int                             not null,
    price                   numeric(10, 2)                  not null,
    amount                  numeric(10, 2)                  not null,

    created_at              timestamp       default now()   not null
);

create table if not exists success_bot.portfolios
(
    user_id                 bigint                          not null,
    instrument_id           bigint                          not null,
    count                   int                             not null,
    average_price           numeric(10, 2)                  not null,

    created_at              timestamp       default now()   not null,
    updated_at              timestamp       default now()   not null,

    unique(user_id, instrument_id)
);

create trigger update_portfolios_updated_at
    before update on success_bot.portfolios
    for each row
    execute function success_bot.update_updated_at();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

drop table if exists success_bot.portfolios;
drop table if exists success_bot.operations;
drop table if exists success_bot.instruments;
drop table if exists success_bot.promocodes;
drop table if exists success_bot.users;
drop function if exists success_bot.update_updated_at();
drop schema if exists success_bot;

-- +goose StatementEnd
