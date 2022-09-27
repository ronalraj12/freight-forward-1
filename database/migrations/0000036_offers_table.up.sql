begin;

alter type image_type add value 'offer';

create table offers(
    id serial primary key,
    title text,
    description text,
    discount int not null,
    image_id int references images(id),
    created_at timestamp default now(),
    archived_at timestamp
);

CREATE UNIQUE INDEX single_active_offer ON offers((archived_at is null)) WHERE archived_at IS NULL;

commit;