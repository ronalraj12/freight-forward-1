BEGIN;

ALTER TYPE image_type
    ADD VALUE 'item';

CREATE TABLE item_images(
  id SERIAL PRIMARY KEY,
  item_id INT REFERENCES items(id),
  image_id INT REFERENCES images(id)
);

COMMIT;
