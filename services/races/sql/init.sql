drop table if exists race_language;
drop table if exists races;
drop table if exists languages;

create table races
(   id int not null
        generated always
        as identity
        primary key
,   name varchar not null
        unique
,   size varchar null
,   speed int null
);

create table languages
(   id int not null
        generated always
        as identity
        primary key
,   name varchar not null
        unique
);

create table race_language
(   id int not null
        generated always
        as identity
        primary key
,   race_id int not null
        references races(id)
,   language_id int not null
        references languages(id)
);

create or replace procedure add_lang
(   lang_name varchar
,    race_id int
) begin atomic

with ins as (
    insert into languages (name) values (lang_name)
    on conflict do nothing
    returning id
), sel as (
    select id from languages
    where name = lang_name
    union
    select id from ins
)
insert into race_language (race_id, language_id)
select race_id, id from sel;

end;
