-- Test schema for skimatik code generator
-- This schema covers all PostgreSQL data types and edge cases

-- Enable UUID extension for UUID support
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create a simple UUID v7-like function for testing
-- Note: This is a simplified version for testing purposes
-- In production, you would use a proper UUID v7 implementation
CREATE OR REPLACE FUNCTION uuid_generate_v7() RETURNS UUID AS $$
BEGIN
    -- Generate a UUID v4 for testing (in production, use proper UUID v7)
    RETURN uuid_generate_v4();
END;
$$ LANGUAGE plpgsql;

-- Users table - Basic table with UUID primary key (UUID v7 required)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_login TIMESTAMP WITH TIME ZONE,
    metadata JSONB,
    age INTEGER,
    balance DECIMAL(10,2),
    profile_picture_url TEXT
);

-- Profiles table - One-to-one relationship with users
CREATE TABLE profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    bio TEXT,
    avatar_url TEXT,
    website_url TEXT,
    location VARCHAR(255),
    birth_date DATE,
    phone VARCHAR(20),
    preferences JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Posts table - One-to-many relationship with users
CREATE TABLE posts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    content TEXT NOT NULL,
    excerpt TEXT,
    slug VARCHAR(500) UNIQUE NOT NULL,
    status VARCHAR(20) DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'archived')),
    published_at TIMESTAMP WITH TIME ZONE,
    view_count INTEGER DEFAULT 0,
    like_count INTEGER DEFAULT 0,
    tags TEXT[] DEFAULT '{}',
    featured_image_url TEXT,
    seo_data JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Comments table - Hierarchical/tree structure with self-referencing FK
CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id UUID REFERENCES comments(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    is_approved BOOLEAN DEFAULT false,
    upvotes INTEGER DEFAULT 0,
    downvotes INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Categories table - Many-to-many relationship with posts (via post_categories)
CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    slug VARCHAR(100) UNIQUE NOT NULL,
    parent_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    sort_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Post-Categories junction table - Many-to-many relationship
CREATE TABLE post_categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    category_id UUID NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(post_id, category_id)
);

-- Files table - Binary data and file metadata
CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    original_filename VARCHAR(255) NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    file_size BIGINT NOT NULL,
    file_hash VARCHAR(64) NOT NULL,
    storage_path TEXT NOT NULL,
    is_public BOOLEAN DEFAULT false,
    download_count INTEGER DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Comprehensive data types table - Tests all PostgreSQL types
CREATE TABLE data_types_test (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v7(),
    
    -- String types
    text_field TEXT,
    varchar_field VARCHAR(255),
    char_field CHAR(10),
    
    -- Numeric types
    smallint_field SMALLINT,
    integer_field INTEGER,
    bigint_field BIGINT,
    decimal_field DECIMAL(10,2),
    numeric_field NUMERIC(15,5),
    real_field REAL,
    double_field DOUBLE PRECISION,
    
    -- Boolean
    boolean_field BOOLEAN,
    
    -- Date/Time types
    date_field DATE,
    time_field TIME,
    timestamp_field TIMESTAMP,
    timestamptz_field TIMESTAMP WITH TIME ZONE,
    interval_field INTERVAL,
    
    -- UUID
    uuid_field UUID,
    
    -- JSON types
    json_field JSON,
    jsonb_field JSONB,
    
    -- Array types
    text_array_field TEXT[],
    integer_array_field INTEGER[],
    uuid_array_field UUID[],
    
    -- Network types
    inet_field INET,
    cidr_field CIDR,
    macaddr_field MACADDR,
    
    -- Other types
    bytea_field BYTEA,
    xml_field XML,
    
    -- Nullable versions (to test pgtype integration)
    nullable_text TEXT,
    nullable_integer INTEGER,
    nullable_boolean BOOLEAN,
    nullable_timestamp TIMESTAMP WITH TIME ZONE,
    nullable_uuid UUID,
    nullable_jsonb JSONB,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Test table with non-UUID primary key (should be rejected by generator)
CREATE TABLE invalid_pk_table (
    id SERIAL PRIMARY KEY,  -- This should cause skimatik to reject the table
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Test table with composite primary key (should be rejected by generator)
CREATE TABLE composite_pk_table (
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (tenant_id, user_id)
);

-- Indexes for performance testing
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_active_created ON users(is_active, created_at);
CREATE INDEX idx_posts_user_status ON posts(user_id, status);
CREATE INDEX idx_posts_published ON posts(published_at) WHERE status = 'published';
CREATE INDEX idx_comments_post_parent ON comments(post_id, parent_id);
CREATE INDEX idx_profiles_user ON profiles(user_id);
CREATE UNIQUE INDEX idx_profiles_user_unique ON profiles(user_id);

-- Create some views for testing (if generator supports views in the future)
CREATE VIEW active_users_view AS
SELECT 
    u.id,
    u.name,
    u.email,
    u.created_at,
    p.bio,
    COUNT(po.id) as post_count
FROM users u
LEFT JOIN profiles p ON u.id = p.user_id
LEFT JOIN posts po ON u.id = po.user_id
WHERE u.is_active = true
GROUP BY u.id, u.name, u.email, u.created_at, p.bio;

-- Create a function for testing (if generator supports functions in the future)
CREATE OR REPLACE FUNCTION get_user_post_count(user_uuid UUID)
RETURNS INTEGER AS $$
BEGIN
    RETURN (
        SELECT COUNT(*)::INTEGER
        FROM posts
        WHERE user_id = user_uuid
    );
END;
$$ LANGUAGE plpgsql;

-- Add some constraints for testing
ALTER TABLE users ADD CONSTRAINT chk_users_age CHECK (age >= 0 AND age <= 150);
ALTER TABLE posts ADD CONSTRAINT chk_posts_counts CHECK (view_count >= 0 AND like_count >= 0);
ALTER TABLE comments ADD CONSTRAINT chk_comments_votes CHECK (upvotes >= 0 AND downvotes >= 0);

-- Schema setup completed
SELECT 'Database schema initialization completed successfully' as status; 