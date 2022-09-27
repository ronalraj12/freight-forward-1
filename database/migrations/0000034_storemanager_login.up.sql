begin;
alter table users
    alter column phone drop not null;

create unique index phone_and_not_archived ON users (phone) where archived_at is null;
create unique index email_and_not_archived ON users (email) where archived_at is null;

ALTER TABLE users
    add column password text default null,
    ADD CONSTRAINT phone_or_email_not_null
        CHECK (
                phone is not null or email is not null
            );
commit;