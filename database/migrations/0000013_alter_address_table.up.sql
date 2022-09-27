alter table address DROP CONSTRAINT address_user_id_type_key;

CREATE UNIQUE INDEX user_id_type_constraint ON address (user_id,type) WHERE archived_at IS NULL;