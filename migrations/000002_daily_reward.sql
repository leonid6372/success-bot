-- +goose Up
-- +goose StatementBegin

alter table success_bot.users add column if not exists daily_reward boolean default false not null;

insert into success_bot.promocodes(available_count, value, bonus_amount) values
    (-1, '🎁 Ежедневная награда', 1000);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

alter table success_bot.users drop column if exists daily_reward;

delete from success_bot.promocodes where id = -1;

-- +goose StatementEnd