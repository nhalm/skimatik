-- Sample data for testing
INSERT INTO users (name, email, bio) VALUES 
    ('Alice Johnson', 'alice@example.com', 'Tech blogger and Go enthusiast'),
    ('Bob Smith', 'bob@example.com', 'Software engineer with a passion for databases'),
    ('Carol Davis', 'carol@example.com', 'Full-stack developer and writer');

INSERT INTO tags (name, description) VALUES
    ('Go', 'Posts about Go programming language'),
    ('PostgreSQL', 'Database-related content'),
    ('Web Development', 'Frontend and backend web development'),
    ('Architecture', 'Software architecture and design patterns');

-- Get user IDs for sample posts
DO $$
DECLARE
    alice_id UUID;
    bob_id UUID;
    carol_id UUID;
    post1_id UUID;
    post2_id UUID;
    go_tag_id UUID;
    pg_tag_id UUID;
    arch_tag_id UUID;
BEGIN
    SELECT id INTO alice_id FROM users WHERE email = 'alice@example.com';
    SELECT id INTO bob_id FROM users WHERE email = 'bob@example.com';
    SELECT id INTO carol_id FROM users WHERE email = 'carol@example.com';
    
    -- Insert sample posts
    INSERT INTO posts (title, content, author_id, is_published, published_at) VALUES 
        ('Getting Started with skimatik', 'skimatik is a database-first code generator that makes building Go applications with PostgreSQL incredibly productive...', alice_id, true, NOW() - INTERVAL '2 days')
        RETURNING id INTO post1_id;
        
    INSERT INTO posts (title, content, author_id, is_published, published_at) VALUES
        ('Multi-Layer Architecture Best Practices', 'When building larger applications, proper layer separation is crucial for maintainability...', bob_id, true, NOW() - INTERVAL '1 day')
        RETURNING id INTO post2_id;
        
    INSERT INTO posts (title, content, author_id, is_published) VALUES
        ('Draft: Advanced PostgreSQL Features', 'This post covers some advanced PostgreSQL features that work great with skimatik...', carol_id, false);
    
    -- Get tag IDs
    SELECT id INTO go_tag_id FROM tags WHERE name = 'Go';
    SELECT id INTO pg_tag_id FROM tags WHERE name = 'PostgreSQL';
    SELECT id INTO arch_tag_id FROM tags WHERE name = 'Architecture';
    
    -- Tag the posts
    INSERT INTO post_tags (post_id, tag_id) VALUES
        (post1_id, go_tag_id),
        (post1_id, pg_tag_id),
        (post2_id, arch_tag_id),
        (post2_id, go_tag_id);
    
    -- Add some comments
    INSERT INTO comments (post_id, author_id, content, is_approved) VALUES
        (post1_id, bob_id, 'Great introduction! Looking forward to trying this out.', true),
        (post1_id, carol_id, 'This looks really promising. Does it support custom queries?', true),
        (post2_id, alice_id, 'Excellent points about layer separation. Very helpful!', true);
END $$;