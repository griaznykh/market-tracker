-- Migration for users and API tokens (UUID generated in application)

create table users (
    id uuid primary key,
    email varchar(255) not null unique,
    password varchar(255) not null,
    created_at timestamp default current_timestamp
);

create table api_token (
    id uuid primary key,
    user_id uuid not null references users(id) on delete cascade,
    token varchar(255) not null unique,
    created_at timestamp default current_timestamp
);

---- create above / drop below ----

drop table users;
drop table api_token;
