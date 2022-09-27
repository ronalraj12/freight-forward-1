begin;
drop table user_session;

alter table address
    add column is_default bool default false not null;

CREATE UNIQUE INDEX default_address_constraint ON address (user_id) WHERE is_default IS TRUE;

commit;
