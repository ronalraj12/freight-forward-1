begin;

ALTER TYPE weekdays RENAME VALUE 'WednesdayThursday' TO 'Wednesday';
ALTER TYPE weekdays ADD VALUE 'Thursday';

commit;