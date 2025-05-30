CREATE TABLE IF NOT EXISTS user_preferences (
                                                user_id UUID NOT NULL,
                                                category INTEGER NOT NULL,
                                                weight INTEGER NOT NULL DEFAULT 0,
                                                PRIMARY KEY (user_id, category)
);

CREATE INDEX idx_user_preferences_user ON user_preferences(user_id);