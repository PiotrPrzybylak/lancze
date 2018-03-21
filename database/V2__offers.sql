CREATE TABLE public.offers (
	id bigserial NOT NULL,
	offer text NULL,
	"date" date NULL,
	place_id bigint NULL,
	CONSTRAINT offers_pk PRIMARY KEY (id)
)
WITH (
	OIDS=FALSE
) ;
