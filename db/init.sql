CREATE TABLE users (
                       id INTEGER PRIMARY KEY,
                       name VARCHAR(30) NOT NULL,
                       surname VARCHAR(30),
                       age SMALLINT,
                       sex BOOLEAN,
                       registration_date DATE NOT NULL,
                       email VARCHAR(40) NOT NULL UNIQUE,
                       phone_number VARCHAR(12) NOT NULL UNIQUE,
                       password_hash TEXT NOT NULL,
                       balance DECIMAL DEFAULT 0.0 NOT NULL,
                       deals_count INTEGER DEFAULT 0 NOT NULL
);

CREATE TABLE user_rating (
                             user_id INTEGER NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
                             rating_count INTEGER DEFAULT 0 NOT NULL,
                             rating_sum INTEGER DEFAULT 0 NOT NULL,
                             rating DECIMAL DEFAULT 0.0 NOT NULL
);

CREATE TABLE user_feedback (
                               id INTEGER PRIMARY KEY,
                               user_recipient_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                               user_writer_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                               comment TEXT,
                               rating SMALLINT NOT NULL
);

CREATE TABLE announcement (
                              id INTEGER PRIMARY KEY,
                              name VARCHAR(100) NOT NULL,
                              description TEXT,
                              user_seller_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                              price DECIMAL NOT NULL,
                              category INTEGER,
                              discount SMALLINT DEFAULT 0 NOT NULL,
                              is_active BOOLEAN DEFAULT TRUE NOT NULL
);

CREATE TABLE announcement_feedback (
                                       id INTEGER PRIMARY KEY,
                                       announcement_recipient_id INTEGER NOT NULL REFERENCES announcement(id) ON DELETE CASCADE,
                                       user_writer_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
                                       comment TEXT,
                                       rating SMALLINT NOT NULL
);

CREATE INDEX idx_user_feedback_recipient ON user_feedback(user_recipient_id);
CREATE INDEX idx_announcement_seller ON announcement(user_seller_id);
CREATE INDEX idx_announcement_feedback_recipient ON announcement_feedback(announcement_recipient_id);