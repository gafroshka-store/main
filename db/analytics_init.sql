CREATE TABLE IF NOT EXISTS user_preferences (
    user_id VARCHAR(64) NOT NULL,
    category INTEGER NOT NULL,
    weight INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, category)
);

-- Optionally, add an index for faster lookups
CREATE INDEX IF NOT EXISTS idx_user_preferences_user ON user_preferences(user_id);