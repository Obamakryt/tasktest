\c test;

CREATE TABLE IF NOT EXISTS users (
id SERIAL PRIMARY KEY,
username text not null unique,
pass_hash text not null,
created_at timestamp default now()
);

CREATE TABLE IF NOT EXISTS documents(
id uuid PRIMARY KEY,
name text not null,
public bool not null,
is_file bool not null,
mime text not null,
json_data JSONB,
file_path text,
created_at timestamp default now(),
own_id INT REFERENCES users(id) ON DELETE CASCADE
    );

CREATE TABLE IF NOT EXISTS document_grants (
 id SERIAL PRIMARY KEY,
 document_id uuid NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
 granted_user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 created_at TIMESTAMP DEFAULT NOW(),
 UNIQUE(document_id, granted_user_id)
);

CREATE TABLE IF NOT EXISTS sessions (
    token_id SERIAL PRIMARY KEY,
    token text not null unique,
    created_at timestamp not null,
    expire_at timestamp not null,
    user_id INT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(user_id)
    );



