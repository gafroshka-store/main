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
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    searching BOOLEAN DEFAULT FALSE NOT NULL
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
CREATE TRIGGER trg_update_announcement_rating_insert
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

SELECT 1;

-- ============ 1. Вставляем 7 пользователей ============

INSERT INTO users (name, surname, day_of_birth, sex, registration_date, email, phone_number, password_hash)
VALUES
    ('Иван',    'Иванов',   '1990-01-01'::timestamptz, TRUE,  CURRENT_DATE, 'ivanov@example.com',    '+79991110001', 'hash1'),
    ('Мария',   'Петрова',  '1985-05-12'::timestamptz, FALSE, CURRENT_DATE, 'petrova@example.com',  '+79991110002', 'hash2'),
    ('Алексей', 'Сидоров',  '1992-07-23'::timestamptz, TRUE,  CURRENT_DATE, 'sidorov@example.com',  '+79991110003', 'hash3'),
    ('Елена',   'Николаева','1988-11-30'::timestamptz, FALSE, CURRENT_DATE, 'nikolaeva@example.com', '+79991110004', 'hash4'),
    ('Дмитрий', 'Кузнецов','1995-03-15'::timestamptz, TRUE,  CURRENT_DATE, 'kuznetsov@example.com','+79991110005', 'hash5'),
    ('Ольга',   'Семенова', '1991-09-02'::timestamptz, FALSE, CURRENT_DATE, 'semenova@example.com', '+79991110006', 'hash6'),
    ('Павел',   'Морозов',  '1987-12-19'::timestamptz, TRUE,  CURRENT_DATE, 'morozov@example.com',  '+79991110007', 'hash7');

-- ============ 2. Вставляем 30 объявлений ============

INSERT INTO announcement (name, description, user_seller_id, price, category, discount)
VALUES
    ('Телефон Samsung Galaxy S21',      'Новый смартфон, 128 ГБ, черный',       (SELECT id FROM users WHERE email = 'ivanov@example.com'),   65000,  1, 5),
    ('Ноутбук Acer Aspire 5',            '15.6" Full HD, Intel i5, 8 ГБ ОЗУ',     (SELECT id FROM users WHERE email = 'petrova@example.com'),  55000,  2, 10),
    ('Телевизор LG OLED55',              'OLED, 55", 4K Ultra HD',               (SELECT id FROM users WHERE email = 'sidorov@example.com'),  120000, 3, 0),
    ('Планшет Apple iPad Air',           '10.9", A14 Bionic, 64 ГБ, серебристый', (SELECT id FROM users WHERE email = 'nikolaeva@example.com'), 45000,  1, 0),
    ('Холодильник Bosch KGN39',           'Двухкамерный, No Frost, серебристый',  (SELECT id FROM users WHERE email = 'kuznetsov@example.com'), 70000,  4, 15),
    ('Микроволновка Panasonic NN-SN686S','23 л, 1000 Вт, стальной',               (SELECT id FROM users WHERE email = 'semenova@example.com'), 11000,  4, 0),
    ('Пылесос Dyson V11',                'Беспроводной, 0.76 л, синий',          (SELECT id FROM users WHERE email = 'morozov@example.com'),  55000,  4, 5),
    ('Наушники Sony WH-1000XM4',         'Шумоподавление, черные',               (SELECT id FROM users WHERE email = 'ivanov@example.com'),   35000,  1, 0),
    ('Колонка JBL Charge 4',             'Bluetooth, водонепроницаемая',         (SELECT id FROM users WHERE email = 'petrova@example.com'),  9000,   5, 0),
    ('Кофемашина DeLonghi ECAM22.110',    'Автоматическая, капучинатор',          (SELECT id FROM users WHERE email = 'sidorov@example.com'),  40000,  4, 10),
    ('Фотоаппарат Canon EOS 250D',        'DSLR, 24 МП, черный',                   (SELECT id FROM users WHERE email = 'nikolaeva@example.com'), 50000,  3, 0),
    ('Велосипед Forward Apache 27.5"',    'Горный, рама 19", черный',             (SELECT id FROM users WHERE email = 'kuznetsov@example.com'), 30000,  6, 0),
    ('Посудомоечная машина Bosch SMS25',  'Полновстраиваемая, 12 комплектов',     (SELECT id FROM users WHERE email = 'semenova@example.com'), 50000,  4, 20),
    ('Клавиатура Logitech MX Keys',      'Беспроводная, подсветка',               (SELECT id FROM users WHERE email = 'morozov@example.com'),  10000,  1, 0),
    ('Монитор Dell U2719D',              '27", WQHD, IPS',                        (SELECT id FROM users WHERE email = 'ivanov@example.com'),   35000,  2, 5),
    ('Принтер HP LaserJet Pro M15w',      'Лазерный, черно-белый, Wi-Fi',          (SELECT id FROM users WHERE email = 'petrova@example.com'),  12000,  4, 0),
    ('Игровая консоль Sony PlayStation 5','825 ГБ, белый',                        (SELECT id FROM users WHERE email = 'sidorov@example.com'),  65000,  7, 0),
    ('Кресло офисное Chairman 420',      'Эргономичное, черное',                 (SELECT id FROM users WHERE email = 'nikolaeva@example.com'), 8000,   8, 0),
    ('Смартфон Xiaomi Redmi Note 10',     '6.43", 64 ГБ, голубой',                (SELECT id FROM users WHERE email = 'kuznetsov@example.com'), 18000,  1, 5),
    ('Умная колонка Яндекс.Станция',     'Wi-Fi, голосовой помощник, белый',      (SELECT id FROM users WHERE email = 'semenova@example.com'),  5000,   5, 0),
    ('Электросамокат Xiaomi Mi M365',    'Макс. скорость 25 км/ч',               (SELECT id FROM users WHERE email = 'morozov@example.com'),  25000,  6, 10),
    ('Бытовой кондиционер LG P09EP',      '9000 BTU, белый',                      (SELECT id FROM users WHERE email = 'ivanov@example.com'),   30000,  4, 5),
    ('Смарт-часы Apple Watch Series 6',   'GPS, 44 мм, космический серый',         (SELECT id FROM users WHERE email = 'petrova@example.com'),  35000,  1, 0),
    ('Телевизор Samsung QLED Q60A',       'QLED, 43", 4K',                        (SELECT id FROM users WHERE email = 'sidorov@example.com'),  70000,  3, 0),
    ('Ноутбук Lenovo ThinkPad X1 Carbon','14", i7, 16 ГБ ОЗУ, черный',            (SELECT id FROM users WHERE email = 'nikolaeva@example.com'), 150000, 2, 10),
    ('Робот-пылесос Xiaomi Roborock S6',  'Лазерная навигация, 5200 мАч',         (SELECT id FROM users WHERE email = 'kuznetsov@example.com'), 40000,  4, 0),
    ('Кофеварка Nespresso Essenza Mini', '19 бар, черная',                       (SELECT id FROM users WHERE email = 'semenova@example.com'), 15000,  4, 5),
    ('Игровое кресло DXRacer Racing',    'Подсветка, поддержка спины, черный',    (SELECT id FROM users WHERE email = 'morozov@example.com'), 20000,  8, 0),
    ('Наушники Bose QuietComfort 35 II', 'Шумоподавление, черные',               (SELECT id FROM users WHERE email = 'ivanov@example.com'),   25000,  1, 0),
    ('Монитор ASUS ROG Strix XG279Q',     '27", 170 Гц, IPS, G-Sync',             (SELECT id FROM users WHERE email = 'petrova@example.com'),  55000,  2, 0),
    ('Проектор Epson EH-TW650',          '3LCD, 2500 лм, Full HD',               (SELECT id FROM users WHERE email = 'sidorov@example.com'),  65000,  3, 0),
    ('Телефон Nokia 3310',               'Винтаж, синий',                       (SELECT id FROM users WHERE email = 'nikolaeva@example.com'),  5000,   1, 0),
    ('Фотоаппарат Nikon D3500',           '24 МП, черный',                        (SELECT id FROM users WHERE email = 'kuznetsov@example.com'),  45000,  3, 0),
    ('Графический планшет Wacom Intuos',  'Pen, черный',                         (SELECT id FROM users WHERE email = 'semenova@example.com'),  12000,  2, 5),
    ('Ноутбук HP Pavilion 15',           '15.6", Ryzen 5, 8 ГБ ОЗУ, серебристый', (SELECT id FROM users WHERE email = 'morozov@example.com'),  65000,  2, 0);

-- ============ 3. Вставляем отзывы (announcement_feedback) ============

-- Для простоты: каждый из 7 пользователей оставляет отзыв на несколько разных объявлений.
-- Мы гарантируем, что комбинация (announcement_recipient_id, user_writer_id) различна.

INSERT INTO announcement_feedback (announcement_recipient_id, user_writer_id, comment, rating)
VALUES
    ((SELECT id FROM announcement WHERE name = 'Телефон Samsung Galaxy S21'),     (SELECT id FROM users WHERE email = 'petrova@example.com'),   'Очень хороший телефон, доволен покупкой',           5),
    ((SELECT id FROM announcement WHERE name = 'Ноутбук Acer Aspire 5'),         (SELECT id FROM users WHERE email = 'ivanov@example.com'),    'Быстрая работа, но шумноват',                        4),
    ((SELECT id FROM announcement WHERE name = 'Телевизор LG OLED55'),           (SELECT id FROM users WHERE email = 'sidorov@example.com'),   'Картинка шикарная, но дорого',                       4),
    ((SELECT id FROM announcement WHERE name = 'Планшет Apple iPad Air'),        (SELECT id FROM users WHERE email = 'nikolaeva@example.com'),'Отличный планшет, экран великолепный',               5),
    ((SELECT id FROM announcement WHERE name = 'Холодильник Bosch KGN39'),       (SELECT id FROM users WHERE email = 'kuznetsov@example.com'),'Тихий, аккуратный, место экономит',                  5),
    ((SELECT id FROM announcement WHERE name = 'Микроволновка Panasonic NN-SN686S'),(SELECT id FROM users WHERE email = 'semenova@example.com'),'Удобная и простая в использовании',                  4),
    ((SELECT id FROM announcement WHERE name = 'Пылесос Dyson V11'),             (SELECT id FROM users WHERE email = 'morozov@example.com'),  'Мощный, но батареи хватает мало',                    3),
    ((SELECT id FROM announcement WHERE name = 'Наушники Sony WH-1000XM4'),      (SELECT id FROM users WHERE email = 'ivanov@example.com'),    'Шумоподавление супер, но дорогие',                  5),
    ((SELECT id FROM announcement WHERE name = 'Колонка JBL Charge 4'),          (SELECT id FROM users WHERE email = 'petrova@example.com'),   'Громкая, компактная, отличный звук',                 5),
    ((SELECT id FROM announcement WHERE name = 'Кофемашина DeLonghi ECAM22.110'), (SELECT id FROM users WHERE email = 'sidorov@example.com'),   'Делает вкусный кофе, но карта мелкая',               4),
    ((SELECT id FROM announcement WHERE name = 'Фотоаппарат Canon EOS 250D'),    (SELECT id FROM users WHERE email = 'nikolaeva@example.com'),'Легкий, удобно держать, отличная картинка',           5),
    ((SELECT id FROM announcement WHERE name = 'Велосипед Forward Apache 27.5"'), (SELECT id FROM users WHERE email = 'kuznetsov@example.com'),'Ходит плавно, но рама могла быть крепче',             4),
    ((SELECT id FROM announcement WHERE name = 'Посудомоечная машина Bosch SMS25'),(SELECT id FROM users WHERE email = 'semenova@example.com'),'Много места, моет отлично',                          5),
    ((SELECT id FROM announcement WHERE name = 'Клавиатура Logitech MX Keys'),   (SELECT id FROM users WHERE email = 'morozov@example.com'),  'Очень удобная, но без подсветки некомфортно ночью',    4),
    ((SELECT id FROM announcement WHERE name = 'Монитор Dell U2719D'),           (SELECT id FROM users WHERE email = 'ivanov@example.com'),    'Яркий, но угол обзора маловат',                      4),
    ((SELECT id FROM announcement WHERE name = 'Принтер HP LaserJet Pro M15w'),  (SELECT id FROM users WHERE email = 'petrova@example.com'),   'Быстрая печать, но шумит сильно',                    3),
    ((SELECT id FROM announcement WHERE name = 'Игровая консоль Sony PlayStation 5'),(SELECT id FROM users WHERE email = 'sidorov@example.com'),'Шикарная консоль, но долго искал в продаже',         5),
    ((SELECT id FROM announcement WHERE name = 'Кресло офисное Chairman 420'),   (SELECT id FROM users WHERE email = 'nikolaeva@example.com'),'Удобно сидеть, но очень дорого',                     3),
    ((SELECT id FROM announcement WHERE name = 'Смартфон Xiaomi Redmi Note 10'), (SELECT id FROM users WHERE email = 'kuznetsov@example.com'),'Хороший экран, но иногда зависает',                  4),
    ((SELECT id FROM announcement WHERE name = 'Умная колонка Яндекс.Станция'),  (SELECT id FROM users WHERE email = 'semenova@example.com'),'Отличный помощник, но голос не всегда понимает',      4),
    ((SELECT id FROM announcement WHERE name = 'Электросамокат Xiaomi Mi M365'), (SELECT id FROM users WHERE email = 'morozov@example.com'),'Очень удобный, но аккумулятор слабоват',               4),
    ((SELECT id FROM announcement WHERE name = 'Бытовой кондиционер LG P09EP'), (SELECT id FROM users WHERE email = 'ivanov@example.com'),    'Прохладно, тишина, но высокая цена',                5),
    ((SELECT id FROM announcement WHERE name = 'Смарт-часы Apple Watch Series 6'),(SELECT id FROM users WHERE email = 'petrova@example.com'),'Стильные, много функций',                             5),
    ((SELECT id FROM announcement WHERE name = 'Телевизор Samsung QLED Q60A'),  (SELECT id FROM users WHERE email = 'sidorov@example.com'),'Красивое изображение, но контраст слабоват',          4),
    ((SELECT id FROM announcement WHERE name = 'Ноутбук Lenovo ThinkPad X1 Carbon'),(SELECT id FROM users WHERE email = 'nikolaeva@example.com'),'Легкий, но стоит очень дорого',                        4),
    ((SELECT id FROM announcement WHERE name = 'Робот-пылесос Xiaomi Roborock S6'),(SELECT id FROM users WHERE email = 'kuznetsov@example.com'),'Убирает хорошо, но иногда застревает',               4),
    ((SELECT id FROM announcement WHERE name = 'Кофеварка Nespresso Essenza Mini'),(SELECT id FROM users WHERE email = 'semenova@example.com'),'Компактная, но капсулы дорогие',                      3),
    ((SELECT id FROM announcement WHERE name = 'Игровое кресло DXRacer Racing'), (SELECT id FROM users WHERE email = 'morozov@example.com'),'Удобное, но жестковато для долгой игры',               4),
    ((SELECT id FROM announcement WHERE name = 'Наушники Bose QuietComfort 35 II'),(SELECT id FROM users WHERE email = 'ivanov@example.com'),'Шумодав отличный, но цена высокая',                   5),
    ((SELECT id FROM announcement WHERE name = 'Монитор ASUS ROG Strix XG279Q'),  (SELECT id FROM users WHERE email = 'petrova@example.com'),'Отзывчивый, но сильно дорогой',                       4);

-- ============ 4. Ещё несколько отзывов для большей насыщенности ============

INSERT INTO announcement_feedback (announcement_recipient_id, user_writer_id, comment, rating)
VALUES
    ((SELECT id FROM announcement WHERE name = 'Проектор Epson EH-TW650'),        (SELECT id FROM users WHERE email = 'sidorov@example.com' ), 'Яркий, но шумит',                                       3),
    ((SELECT id FROM announcement WHERE name = 'Телефон Nokia 3310'),             (SELECT id FROM users WHERE email = 'ivanov@example.com'),              'Ностальгия, долго не разряжается',                          5),
    ((SELECT id FROM announcement WHERE name = 'Фотоаппарат Nikon D3500'),        (SELECT id FROM users WHERE email = 'petrova@example.com'),             'Простой в обращении, отличное фото',                        4),
    ((SELECT id FROM announcement WHERE name = 'Графический планшет Wacom Intuos'),(SELECT id FROM users WHERE email = 'sidorov@example.com'),             'Рисовать удобно, но чуть тяжеловат',                        4),
    ((SELECT id FROM announcement WHERE name = 'Ноутбук HP Pavilion 15'),         (SELECT id FROM users WHERE email = 'nikolaeva@example.com'),           'Хороший ноут, но греется',                                 3);

SELECT 2;