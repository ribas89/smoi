CREATE TABLE IF NOT EXISTS users (
 id TEXT PRIMARY KEY,
 oidc_issuer TEXT NOT NULL,
 oidc_subject TEXT NOT NULL,
 username TEXT NOT NULL DEFAULT '',
 email TEXT NOT NULL DEFAULT '',
 display_name TEXT NOT NULL DEFAULT '',
 avatar_url TEXT NOT NULL DEFAULT '',
 role TEXT NOT NULL DEFAULT 'user' CHECK(role IN ('user','admin')),
 created_at INTEGER NOT NULL,
 updated_at INTEGER NOT NULL,
 last_login_at INTEGER NOT NULL,
 UNIQUE(oidc_issuer, oidc_subject)
);
CREATE TABLE IF NOT EXISTS sessions (
 id TEXT PRIMARY KEY,
 user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 session_token_hash TEXT NOT NULL UNIQUE,
 expires_at INTEGER NOT NULL,
 created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE TABLE IF NOT EXISTS oidc_flows (
 state_hash TEXT PRIMARY KEY,
 nonce TEXT NOT NULL,
 verifier TEXT NOT NULL,
 expires_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS login_exchanges (
 token_hash TEXT PRIMARY KEY,
 user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 expires_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS conversations (
 id TEXT PRIMARY KEY,
 user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 title TEXT NOT NULL DEFAULT 'New chat',
 model TEXT NOT NULL DEFAULT '',
 created_at INTEGER NOT NULL,
 updated_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_conversations_user_updated ON conversations(user_id, updated_at DESC);
CREATE TABLE IF NOT EXISTS messages (
 id TEXT PRIMARY KEY,
 conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
 role TEXT NOT NULL CHECK(role IN ('system','user','assistant')),
 content TEXT NOT NULL DEFAULT '',
 client_request_id TEXT UNIQUE,
 created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id, created_at, id);
CREATE TABLE IF NOT EXISTS attachments (
 id TEXT PRIMARY KEY,
 user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
 message_id TEXT REFERENCES messages(id) ON DELETE CASCADE,
 filename TEXT NOT NULL,
 mime_type TEXT NOT NULL,
 size INTEGER NOT NULL,
 storage_reference TEXT NOT NULL,
 created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_attachments_message ON attachments(message_id);
CREATE INDEX IF NOT EXISTS idx_attachments_conversation ON attachments(conversation_id);
CREATE INDEX IF NOT EXISTS idx_attachments_pending ON attachments(created_at) WHERE message_id IS NULL;
CREATE TABLE IF NOT EXISTS generation_jobs (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    user_message_id TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    assistant_message_id TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    model TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK(status IN ('pending', 'running', 'completed', 'stopped', 'failed')),
    error TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_generation_jobs_conversation ON generation_jobs(conversation_id, created_at, id);
CREATE INDEX IF NOT EXISTS idx_generation_jobs_pending ON generation_jobs(status, conversation_id, created_at);
