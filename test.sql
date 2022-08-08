CREATE TABLE users (
	id SERIAL PRIMARY KEY,
	username VARCHAR(20) NOT NULL UNIQUE,
	password TEXT);

CREATE TABLE authorizations (
	username VARCHAR(20) REFERENCES users (username),
	authorized BOOLEAN DEFAULT false,
	login_time TIMESTAMP,
	logout_time TIMESTAMP);

CREATE TABLE variants (
	id SERIAL PRIMARY KEY,
	name VARCHAR(40) NOT NULL);

CREATE TABLE test_start (
	id SERIAL PRIMARY KEY,
	user_id INTEGER REFERENCES users (id),
	variant_id INTEGER REFERENCES variants (id),
	start_time TIMESTAMP);

CREATE TABLE questions (
	id SERIAL PRIMARY KEY,
	variant_id INTEGER REFERENCES variants (id),
	question VARCHAR(512));

CREATE TABLE answers_variants (
	id SERIAL PRIMARY KEY,
	question_id INTEGER REFERENCES questions (id),
	answer VARCHAR(256),
	correct BOOLEAN);

CREATE TABLE answers (
	id SERIAL PRIMARY KEY,
	test_id INTEGER REFERENCES test_start (id),
	answer_id INTEGER REFERENCES answers_variants (id));

CREATE TABLE results (
	id SERIAL PRIMARY KEY,
	test_id INTEGER REFERENCES test_start (id),
	percent INTEGER);

CREATE EXTENSION pgcrypto;
DO $user$
BEGIN
	IF NOT EXISTS (SELECT * FROM pg_user WHERE usename = 'tester') THEN
		CREATE USER tester WITH LOGIN PASSWORD 'password';
	END IF;
END; $user$;

DO $log$
BEGIN
EXECUTE 'ALTER DATABASE ' || current_database() || ' SET log_statement = ''all''';
END; $log$;

DO $priv$
BEGIN
EXECUTE 'GRANT ALL PRIVILEGES ON DATABASE ' || current_database() || ' TO tester';
EXECUTE 'GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO tester';
END; $priv$;
