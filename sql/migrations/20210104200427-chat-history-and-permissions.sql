-- +migrate Up
alter table chat_history
    add column message_id text not null default '';

alter table discord_server
    add column moderation_mute_enabled bool default false not null;
alter table discord_server
    add column notify_me_role_enabled bool default false not null;

create table role_permission
(
    id                      bigserial primary key,
    guild_id                text               not null,
    role_id                 text               not null,
    post_links              bool default true  not null,
    moderation_mute_member  bool default false not null,
    roll_dice               bool default true  not null,
    flip_coin               bool default true  not null,
    random_image            bool default true  not null,
    use_custom_commands     bool default true  not null,
    manage_custom_commands  bool default false not null,
    ignore_command_throttle bool default false not null,
    play_songs              bool default true  not null,
    play_lists              bool default true  not null,
    skip_songs              bool default false not null
);

create unique index idx_guild_role on role_permission (guild_id, role_id);

create table muted_members
(
    id         bigserial primary key,
    user_id    text                    not null,
    guild_id   text                    not null,
    created_at timestamp default now() not null,
    expires_at timestamp default null
);

create unique index idx_user_guild on muted_members (user_id, guild_id);

-- +migrate Down
alter table chat_history
    drop column message_id;
alter table discord_server
    drop column moderation_mute_enabled;
alter table discord_server
    drop column notify_me_role_enabled;
drop table role_permission;
drop table muted_members;
