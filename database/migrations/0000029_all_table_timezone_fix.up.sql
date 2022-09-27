begin;

truncate address,categories,chat,disputed_orders,
    fcm_token,images,item_images,items,location,
    order_items,order_otp,orders,scheduled_ordered_items,
    scheduled_orders,scheduled_orders_days,user_permission,
    users restart identity cascade;


alter table orders
    alter column delivery_time type timestamp with time zone,
    alter column created_at type timestamp with time zone,
    alter column archived_at type timestamp with time zone,
    alter column updated_at type timestamp with time zone;

alter table address
    alter column created_at type timestamp with time zone,
    alter column updated_at type timestamp with time zone,
    alter column archived_at type timestamp with time zone;

alter table users
    alter column created_at type timestamp with time zone,
    alter column updated_at type timestamp with time zone,
    alter column archived_at type timestamp with time zone;

alter table scheduled_orders_days
    alter column created_at type timestamp with time zone,
    alter column updated_at type timestamp with time zone,
    alter column archived_at type timestamp with time zone;

alter table scheduled_orders
    alter column created_at type timestamp with time zone,
    alter column updated_at type timestamp with time zone,
    alter column archived_at type timestamp with time zone;

alter table scheduled_ordered_items
    alter column created_at type timestamp with time zone,
    alter column updated_at type timestamp with time zone,
    alter column archived_at type timestamp with time zone;

alter table location
    alter column updated_at type timestamp with time zone;

alter table disputed_orders
    drop column resolved_at,
    add column resolved_at timestamp with time zone,
    alter column disputed_at type timestamp with time zone;

alter table fcm_token
    alter column updated_at type timestamp with time zone,
    alter column created_at type timestamp with time zone;

alter table images
    alter column created_at type timestamp with time zone,
    alter column updated_at type timestamp with time zone,
    alter column archived_at type timestamp with time zone;

alter table items
    alter column created_at type timestamp with time zone,
    alter column updated_at type timestamp with time zone,
    alter column archived_at type timestamp with time zone;


commit;