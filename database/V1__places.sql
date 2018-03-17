CREATE TABLE public.places (
	id bigserial NOT NULL,
	"name" text NOT NULL,
	CONSTRAINT places_pk PRIMARY KEY (id)
)
WITH (
	OIDS=FALSE
) ;
