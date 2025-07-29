-- Blog Application Database Initialization
-- Sets up required extensions and functions for skimatik

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

-- Initialization completed
SELECT 'Blog database initialization completed successfully' as status; 