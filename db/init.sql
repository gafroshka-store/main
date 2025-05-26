CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(30) NOT NULL,
    surname VARCHAR(30),
    day_of_birth timestamptz,
    sex BOOLEAN,
    registration_date DATE NOT NULL,
    email VARCHAR(40) NOT NULL UNIQUE,
    phone_number VARCHAR(12) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    balance DECIMAL DEFAULT 0.0 NOT NULL,
    deals_count INTEGER DEFAULT 0 NOT NULL,
    rating FLOAT DEFAULT 0.0 NOT NULL,
    rating_count INTEGER DEFAULT 0 NOT NULL
);

CREATE TABLE user_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_recipient_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_writer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    comment TEXT,
    rating SMALLINT NOT NULL
);

CREATE TABLE announcement (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    user_seller_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    price DECIMAL NOT NULL CHECK (price >= 0),
    category INTEGER CHECK (discount BETWEEN 0 AND 100),
    discount SMALLINT DEFAULT 0 NOT NULL,
    is_active BOOLEAN DEFAULT TRUE NOT NULL,
    rating FLOAT DEFAULT 0.0,
    rating_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE announcement_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    announcement_recipient_id UUID NOT NULL REFERENCES announcement(id) ON DELETE CASCADE,
    user_writer_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    comment TEXT,
    rating SMALLINT NOT NULL
);

CREATE TABLE shopping_cart (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    announcement_id UUID NOT NULL REFERENCES announcement(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, announcement_id)
);


CREATE INDEX idx_user_feedback_recipient ON user_feedback(user_recipient_id);
CREATE INDEX idx_announcement_seller ON announcement(user_seller_id);
CREATE INDEX idx_announcement_feedback_recipient ON announcement_feedback(announcement_recipient_id);
CREATE INDEX idx_cart_user_id ON shopping_cart(user_id);
CREATE INDEX idx_cart_announcement_id ON shopping_cart(announcement_id);

-- Функция для обновления рейтинга и количества отзывов
CREATE OR REPLACE FUNCTION update_announcement_rating()
RETURNS TRIGGER AS $$
BEGIN
  UPDATE announcement
  SET 
    rating_count = rating_count + 1,
    rating = (
      (rating * rating_count + NEW.rating) / (rating_count + 1)
    )
  WHERE id = NEW.announcement_recipient_id;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Триггер, который вызывает эту функцию после вставки
CREATE TRIGGER trg_update_announcement_rating
AFTER INSERT ON announcement_feedback
FOR EACH ROW
EXECUTE FUNCTION update_announcement_rating();

ALTER TABLE announcement_feedback
ADD CONSTRAINT uniq_announcement_writer
  UNIQUE (announcement_recipient_id, user_writer_id);
