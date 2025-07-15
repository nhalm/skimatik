-- Migration 001: Test Data for Query-Based Generation Testing
-- This migration adds test data to support comprehensive testing

-- Insert test users (using specific UUIDs for predictable testing)
INSERT INTO users (id, name, email, password_hash, is_active, age, balance, metadata) VALUES
    (uuid_generate_v7(), 'John Doe', 'john@example.com', 'hashed_password_1', true, 30, 1000.50, '{"role": "admin", "preferences": {"theme": "dark"}}'),
    (uuid_generate_v7(), 'Jane Smith', 'jane@example.com', 'hashed_password_2', true, 25, 750.25, '{"role": "user", "preferences": {"theme": "light"}}'),
    (uuid_generate_v7(), 'Bob Johnson', 'bob@example.com', 'hashed_password_3', false, 35, 0.00, '{"role": "user", "suspended": true}'),
    (uuid_generate_v7(), 'Alice Brown', 'alice@example.com', 'hashed_password_4', true, 28, 2500.75, '{"role": "moderator", "verified": true}');

-- Insert profiles for users
INSERT INTO profiles (id, user_id, bio, location, preferences)
SELECT 
    uuid_generate_v7(),
    u.id,
    'Bio for ' || u.name,
    CASE 
        WHEN u.name = 'John Doe' THEN 'New York, NY'
        WHEN u.name = 'Jane Smith' THEN 'San Francisco, CA'
        WHEN u.name = 'Bob Johnson' THEN 'Chicago, IL'
        ELSE 'Boston, MA'
    END,
    '{"notifications": true, "privacy": "public"}'
FROM users u;

-- Insert categories
INSERT INTO categories (id, name, description, slug, sort_order) VALUES
    (uuid_generate_v7(), 'Technology', 'Tech-related posts', 'technology', 1),
    (uuid_generate_v7(), 'Programming', 'Programming tutorials and tips', 'programming', 2),
    (uuid_generate_v7(), 'Web Development', 'Web development topics', 'web-development', 3),
    (uuid_generate_v7(), 'Database', 'Database design and optimization', 'database', 4);

-- Insert posts
INSERT INTO posts (id, user_id, title, content, excerpt, slug, status, published_at, view_count, like_count, tags)
SELECT 
    uuid_generate_v7(),
    u.id,
    'Sample Post by ' || u.name,
    'This is the content of a sample post by ' || u.name || '. It contains multiple paragraphs and demonstrates the text field capabilities.',
    'Sample excerpt for post by ' || u.name,
    'sample-post-' || LOWER(REPLACE(u.name, ' ', '-')),
    CASE WHEN u.is_active THEN 'published' ELSE 'draft' END,
    CASE WHEN u.is_active THEN NOW() - INTERVAL '1 day' ELSE NULL END,
    FLOOR(RANDOM() * 1000)::INTEGER,
    FLOOR(RANDOM() * 100)::INTEGER,
    ARRAY['sample', 'test', 'demo']
FROM users u;

-- Insert comments
INSERT INTO comments (id, post_id, user_id, content, is_approved, upvotes)
SELECT 
    uuid_generate_v7(),
    p.id,
    u.id,
    'This is a comment on ' || p.title || ' by ' || u.name,
    true,
    FLOOR(RANDOM() * 20)::INTEGER
FROM posts p
CROSS JOIN users u
WHERE u.is_active = true
LIMIT 10;

-- Insert post-category relationships
INSERT INTO post_categories (id, post_id, category_id)
SELECT 
    uuid_generate_v7(),
    p.id,
    c.id
FROM posts p
CROSS JOIN categories c
WHERE RANDOM() < 0.5  -- Randomly assign categories
LIMIT 8;

-- Insert test data for data_types_test table
INSERT INTO data_types_test (
    id, text_field, varchar_field, char_field, smallint_field, integer_field, bigint_field,
    decimal_field, numeric_field, real_field, double_field, boolean_field, date_field,
    time_field, timestamp_field, timestamptz_field, interval_field, uuid_field,
    json_field, jsonb_field, text_array_field, integer_array_field, uuid_array_field,
    inet_field, cidr_field, macaddr_field, nullable_text, nullable_integer, nullable_boolean
) VALUES (
    uuid_generate_v7(),
    'Sample text field with unicode: 你好世界 🌍',
    'VARCHAR field',
    'CHAR      ',  -- Note: CHAR(10) pads with spaces
    32767,
    2147483647,
    9223372036854775807,
    12345.67,
    123456.78901,
    3.14159,
    2.718281828459045,
    true,
    '2024-01-15',
    '14:30:00',
    '2024-01-15 14:30:00',
    '2024-01-15 14:30:00+00',
    '1 year 2 months 3 days 4 hours 5 minutes 6 seconds',
    uuid_generate_v7(),
    '{"key": "value", "number": 42}',
    '{"nested": {"array": [1, 2, 3]}, "boolean": true}',
    ARRAY['one', 'two', 'three'],
    ARRAY[1, 2, 3, 4, 5],
    ARRAY[uuid_generate_v7(), uuid_generate_v7()],
    '192.168.1.1',
    '192.168.1.0/24',
    '08:00:2b:01:02:03',
    'Nullable text value',
    42,
    false
);

-- Insert a row with mostly NULL values to test nullable handling
INSERT INTO data_types_test (id, text_field, nullable_text) VALUES (
    uuid_generate_v7(),
    'Row with mostly NULL values',
    NULL
);

-- Final verification queries for migration completion
SELECT 'Test data migration completed successfully' as status;
SELECT 'Total users: ' || COUNT(*)::text as user_count FROM users;
SELECT 'Total posts: ' || COUNT(*)::text as post_count FROM posts;
SELECT 'Total comments: ' || COUNT(*)::text as comment_count FROM comments;
SELECT 'Total categories: ' || COUNT(*)::text as category_count FROM categories; 