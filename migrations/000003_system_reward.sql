-- +goose Up
-- +goose StatementBegin

insert into success_bot.promocodes(id, available_count, value, bonus_amount) values
    (-2, -1, '🤝 Помощь в разработке', 10000),
    (-3, -1, '🤝 Помощь в разработке', 20000),
    (-4, -1, '🤝 Помощь в разработке', 30000);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

delete from success_bot.promocodes where id = -2;
delete from success_bot.promocodes where id = -3;
delete from success_bot.promocodes where id = -4;

-- +goose StatementEnd