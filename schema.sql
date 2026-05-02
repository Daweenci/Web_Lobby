-- schema.sql - Database schema for the lobby

CREATE TABLE IF NOT EXISTS players (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_login DATETIME
);


CREATE TABLE IF NOT EXISTS friend_requests (
    sender_id TEXT NOT NULL,
    receiver_id TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    status TEXT DEFAULT 'pending',

    PRIMARY KEY (sender_id, receiver_id),

    FOREIGN KEY (sender_id) REFERENCES players(id),
    FOREIGN KEY (receiver_id) REFERENCES players(id),

    CHECK (sender_id != receiver_id)
);

CREATE INDEX IF NOT EXISTS idx_friend_request_receiver_id ON friend_requests(receiver_id);


CREATE TABLE IF NOT EXISTS friend_lists (
    first_player_id TEXT NOT NULL,
    second_player_id TEXT NOT NULL,

    PRIMARY KEY (first_player_id, second_player_id),

    FOREIGN KEY (first_player_id) REFERENCES players(id),
    FOREIGN KEY (second_player_id) REFERENCES players(id),

    CHECK (first_player_id < second_player_id)
);

CREATE INDEX IF NOT EXISTS idx_first_player_id ON friend_lists(first_player_id);
CREATE INDEX IF NOT EXISTS idx_second_player_id ON friend_lists(second_player_id);


CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,

    first_player_id TEXT NOT NULL,
    second_player_id TEXT NOT NULL,

    sender_id TEXT NOT NULL,
    content TEXT NOT NULL,

    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (first_player_id, second_player_id)
        REFERENCES friend_lists(first_player_id, second_player_id),

    FOREIGN KEY (sender_id)
        REFERENCES players(id),

    CHECK (sender_id = first_player_id OR sender_id = second_player_id),
    CHECK (first_player_id < second_player_id)
);

CREATE INDEX IF NOT EXISTS idx_messages_chat ON messages(first_player_id, second_player_id, created_at);


CREATE TABLE IF NOT EXISTS bot_friends (
    id TEXT NOT NULL,
    name TEXT NOT NULL,
    character_prompt TEXT NOT NULL,

    PRIMARY KEY (id)

);


CREATE TABLE IF NOT EXISTS bot_friend_lists (
    player_id TEXT NOT NULL,
    bot_id TEXT NOT NULL,
    chat_history_summary TEXT,
    PRIMARY KEY (player_id, bot_id),

    FOREIGN KEY (player_id) REFERENCES players(id),
    FOREIGN KEY (bot_id) REFERENCES bot_friends(id)
);

CREATE INDEX IF NOT EXISTS idx_bot_friend_lists_player ON bot_friend_lists(player_id);


CREATE TABLE IF NOT EXISTS bot_private_messages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    player_id TEXT NOT NULL,
    bot_id TEXT NOT NULL,
    sender_type TEXT NOT NULL, -- 'player' or 'bot'
    content TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (player_id, bot_id) REFERENCES bot_friend_lists(player_id, bot_id)
);

CREATE INDEX IF NOT EXISTS idx_bot_private_messages ON bot_private_messages(player_id, bot_id, created_at);
