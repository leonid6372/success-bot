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

insert into success_bot.instruments(ticker, name) values
    ('SBER@MISX', '🏦 Сбер Банк'), ('T@MISX', '🏦 Т-Технологии'),
    ('LKOH@MISX', '⛽️ ЛУКОЙЛ'), ('GAZP@MISX', '🔥 Газпром'),
    ('VTBR@MISX', '🏦 Банк ВТБ'), ('GMKN@MISX', '🪨 Норильский никель'),
    ('YDEX@MISX', '🔎 Яндекс'), ('X5@MISX', '🛒 Корп. Центр Икс 5'),
    ('NVTK@MISX', '🔥 НОВАТЭК'), ('OZON@MISX', '📦 МКПАО Озон'),
    ('ROSN@MISX', '⛽️ Роснефть'), ('MOEX@MISX', '💵 Московская Биржа'),
    ('PLZL@MISX', '🪨 Полюс'), ('AQUA@MISX', '🐟 ИНАРКТИКА'),
    ('SNGS@MISX', '🏭 Сургутнефтегаз'), ('TATN@MISX', '⛽️ Татнефть'),
    ('AFLT@MISX', '✈️ Аэрофлот'), ('PIKK@MISX', '🏗 ПИК СЗ (ПАО)'),
    ('NLMK@MISX', '🪨 НЛМК'), ('MAGN@MISX', '🪨 Магнитогор. металлург. комбинат'),
    ('AFKS@MISX', '💵 АФК Система'), ('RUAL@MISX', '🪨 РУСАЛ'),
    ('CHMF@MISX', '🪨 Северсталь'), ('DOMRF@MISX', '🏦 ПАО ДОМ.РФ'),
    ('SMLT@MISX', '🏗 ГК Самолет'), ('HEAD@MISX', '👔 Хэдхантер'),
    ('IRAO@MISX', '💡 Интер РАО ЕЭС'), ('MTSS@MISX', '🥚 МТС'),
    ('MDMG@MISX', '🤱 Мать и дитя'), ('EUTR@MISX', '🚛 ЕвроТранс'),
    ('MTLR@MISX', '🏭 Мечел'), ('UPRO@MISX', '💡 Юнипро'),
    ('ASTR@MISX', '💾 Группа Астра'), ('CBOM@MISX', '🏦 МКБ'),
    ('POSI@MISX', '💾 Группа Позитив'), ('SPBE@MISX', '💵 СПБ Биржа'),
    ('BSPB@MISX', '🏦 Банк Санкт-Петербург'), ('FLOT@MISX', '⚓️ Совкомфлот'),
    ('BELU@MISX', '🥃 Novabev Group'), ('HYDR@MISX', '🌊 РусГидро'),
    ('IVAT@MISX', '💾 IVA Technologies'), ('CNRU@MISX', '🏡 Циан'),
    ('FIXR@MISX', '🛒 ПАО "Фикс Прайс"');


create table if not exists success_bot.operations
(
    id                      bigserial       primary key,

    user_id                 bigint                          not null,
    instrument_id           bigint                          not null, -- id value from instruments table or promocodes table up to type
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
