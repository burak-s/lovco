CREATE TYPE leftover_type AS ENUM ('food', 'electronic', 'clothing', 'other');

CREATE TABLE IF NOT EXISTS leftover (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	owner_id UUID NOT NULL,
	name VARCHAR(255) NOT NULL,
	description TEXT NOT NULL,
	type leftover_type NOT NULL DEFAULT 'other',
	image_url VARCHAR(255) NOT NULL,
	longitude DOUBLE PRECISION NOT NULL,
	latitude DOUBLE PRECISION NOT NULL,
	street VARCHAR(255) NOT NULL,
	district VARCHAR(255) NOT NULL,
	city VARCHAR(255) NOT NULL,
	province VARCHAR(255) NOT NULL,
	state VARCHAR(255),
	country VARCHAR(255) NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	deleted_at TIMESTAMP NULL
);