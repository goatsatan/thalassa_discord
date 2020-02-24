-- +migrate Up
create table if not exists discord_server
(
    guild_id                  text primary key,
    guild_name                text not null,
    link_removal_enabled      bool not null,
    music_enabled             bool not null,
    custom_commands_enabled   bool not null,
    dice_roll_enabled         bool not null,
    prefix_command            text not null default '.',
    music_text_channel_id     text          default null,
    music_voice_channel_id    text          default null,
    music_volume              real not null default 0.5,
    announce_songs            bool not null,
    throttle_commands_enabled bool not null,
    throttle_commands_seconds bigint        default 10,
    welcome_message_enabled   bool not null,
    welcome_message           text
);

create table song
(
    id                  text primary key,
    platform            text default null,
    song_name           text not null,
    description         text,
    url                 text not null,
    duration_in_seconds int,
    is_stream           bool not null,
    thumbnail_url       text,
    artist              text,
    album               text,
    track               text
);

create table song_request
(
    id                   bigserial primary key,
    song_id              text references song,
    song_name            text                     not null,
    requested_by_user_id text                     not null,
    username_at_time     text                     not null,
    guild_id             text                     not null references discord_server,
    guild_name_at_time   text                     not null,
    requested_at         timestamp with time zone not null default now(),
    played_at            timestamp with time zone          default null,
    played               bool                     not null default false
);

create table chat_history
(
    id                         bigserial primary key,
    user_id                    text not null,
    username_at_time           text not null,
    guild_id                   text not null references discord_server,
    guild_name_at_time         text not null,
    guild_channel_id           text not null,
    guild_channel_name_at_time text not null
);

create table custom_command
(
    id               bigserial primary key,
    added_by_user_id text                not null,
    guild_id         text                not null references discord_server,
    command_name     text                not null,
    message          text                not null,
    created_at       time with time zone not null default now(),
    updated_at       time with time zone
);

-- +migrate Down
drop table song_request;
drop table chat_history;
drop table custom_command;
-- drop table discord_server;