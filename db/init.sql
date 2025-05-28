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

-- Функция для обновления рейтинга и количества отзывов при удалении
CREATE OR REPLACE FUNCTION update_announcement_rating_on_delete()
RETURNS TRIGGER AS $$
DECLARE
  new_rating FLOAT;
  new_count INTEGER;
BEGIN
  SELECT COUNT(*), COALESCE(AVG(rating), 0)
  INTO new_count, new_rating
  FROM announcement_feedback
  WHERE announcement_recipient_id = OLD.announcement_recipient_id;

  UPDATE announcement
  SET
    rating_count = new_count,
    rating = new_rating
  WHERE id = OLD.announcement_recipient_id;

  RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_update_announcement_rating_on_delete
AFTER DELETE ON announcement_feedback
FOR EACH ROW
EXECUTE FUNCTION update_announcement_rating_on_delete();

CREATE OR REPLACE FUNCTION update_announcement_rating_on_feedback_update()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE announcement
    SET rating = (
            SELECT COALESCE(AVG(rating), 0)
            FROM announcement_feedback
            WHERE announcement_recipient_id = NEW.announcement_recipient_id
        ),
        rating_count = (
            SELECT COUNT(*)
            FROM announcement_feedback
            WHERE announcement_recipient_id = NEW.announcement_recipient_id
        )
    WHERE id = NEW.announcement_recipient_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_update_announcement_rating ON announcement_feedback;

CREATE TRIGGER trg_update_announcement_rating
AFTER UPDATE OF rating ON announcement_feedback
FOR EACH ROW
EXECUTE PROCEDURE update_announcement_rating_on_feedback_update();

-- Триггер, который вызывает эту функцию после вставки
CREATE TRIGGER trg_update_announcement_rating
AFTER INSERT ON announcement_feedback
FOR EACH ROW
EXECUTE FUNCTION update_announcement_rating();

ALTER TABLE announcement_feedback
ADD CONSTRAINT uniq_announcement_writer
  UNIQUE (announcement_recipient_id, user_writer_id);

-- Функция для обновления рейтинга и количества отзывов пользователя при вставке
CREATE OR REPLACE FUNCTION update_user_rating_on_insert()
RETURNS TRIGGER AS $$
BEGIN
  UPDATE users
  SET 
    rating_count = rating_count + 1,
    rating = (
      (rating * rating_count + NEW.rating) / (rating_count + 1)
    )
  WHERE id = NEW.user_recipient_id;

  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_update_user_rating_on_insert
AFTER INSERT ON user_feedback
FOR EACH ROW
EXECUTE FUNCTION update_user_rating_on_insert();

-- Функция для обновления рейтинга и количества отзывов пользователя при удалении
CREATE OR REPLACE FUNCTION update_user_rating_on_delete()
RETURNS TRIGGER AS $$
DECLARE
  new_rating FLOAT;
  new_count INTEGER;
BEGIN
  SELECT COUNT(*), COALESCE(AVG(rating), 0)
  INTO new_count, new_rating
  FROM user_feedback
  WHERE user_recipient_id = OLD.user_recipient_id;

  UPDATE users
  SET
    rating_count = new_count,
    rating = new_rating
  WHERE id = OLD.user_recipient_id;

  RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_update_user_rating_on_delete
AFTER DELETE ON user_feedback
FOR EACH ROW
EXECUTE FUNCTION update_user_rating_on_delete();

-- Функция для обновления рейтинга и количества отзывов пользователя при обновлении рейтинга
CREATE OR REPLACE FUNCTION update_user_rating_on_update()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE users
    SET rating = (
            SELECT COALESCE(AVG(rating), 0)
            FROM user_feedback
            WHERE user_recipient_id = NEW.user_recipient_id
        ),
        rating_count = (
            SELECT COUNT(*)
            FROM user_feedback
            WHERE user_recipient_id = NEW.user_recipient_id
        )
    WHERE id = NEW.user_recipient_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_update_user_rating_on_update
AFTER UPDATE OF rating ON user_feedback
FOR EACH ROW
EXECUTE PROCEDURE update_user_rating_on_update();

ALTER TABLE user_feedback
ADD CONSTRAINT uniq_user_feedback_writer
  UNIQUE (user_recipient_id, user_writer_id);

CREATE UNIQUE INDEX idx_user_feedback_unique_pair
ON user_feedback (user_recipient_id, user_writer_id);
