-- sql/schema/001_initial.sql
-- Create enums
CREATE TYPE user_role AS ENUM ('USER', 'ADMIN', 'MODERATOR');
CREATE TYPE user_status AS ENUM ('ACTIVE', 'SLEEP', 'OFFLINE', 'BANNED', 'PENDING', 'IDLE', 'INVISIBLE');
CREATE TYPE theme AS ENUM ('LIGHT', 'DARK', 'SYSTEM');
CREATE TYPE language AS ENUM ('En', 'Es', 'Fr', 'De', 'Zh', 'Ja', 'Ru', 'Bg');
CREATE TYPE text_size AS ENUM ('BIG', 'MEDIUM', 'SMALL');
CREATE TYPE role_colors AS ENUM ('NEXTTONAME', 'INNAME', 'DONTSHOW', 'SYNCPROFILECOLORS');
CREATE TYPE motion AS ENUM ('REDUCEDMOTION', 'SYNCWITHDEVICE', 'DONTPLAYGIFWHENPOSSIBLESHOW', 'PLAYEMOJIES');
CREATE TYPE stickers_animation AS ENUM ('ALWAYS', 'ONINTERACTION', 'NEVER');
CREATE TYPE messages_allowance AS ENUM ('ALLMSG', 'UNREADMAS', 'HIDE');

-- Users table
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    first_name TEXT,
    last_name TEXT,
    user_name TEXT,
    image_url TEXT DEFAULT 'https://48htuluf59.ufs.sh/f/1NvBfFppWcZeWF2WCCi3zDay6IgjQLVNYHEhKiCJ8OeGwTon',
    phone_number TEXT,
    role user_role DEFAULT 'USER',
    status user_status DEFAULT 'ACTIVE',
    completed_tutorial BOOLEAN DEFAULT false,
    about_me TEXT,
    note TEXT DEFAULT '',
    disable_account BOOLEAN DEFAULT false,
    delete_account BOOLEAN DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- User friends table
CREATE TABLE user_friends (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    friend_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_name TEXT NOT NULL,
    first_name TEXT,
    last_name TEXT,
    image_url TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, friend_id)
);

-- Settings table
CREATE TABLE settings (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id TEXT UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    theme theme DEFAULT 'LIGHT',
    language language DEFAULT 'En',
    text_size text_size DEFAULT 'MEDIUM',
    zoom_level TEXT DEFAULT '100',
    font_style TEXT DEFAULT 'default',
    messages_allowance messages_allowance DEFAULT 'ALLMSG',
    show_role_colors role_colors DEFAULT 'NEXTTONAME',
    motion motion DEFAULT 'DONTPLAYGIFWHENPOSSIBLESHOW',
    stickers_animation stickers_animation DEFAULT 'ALWAYS',
    enabled_location_tracking BOOLEAN DEFAULT false,
    allow_city_stat_data_usage BOOLEAN DEFAULT true,
    allow_data_personalization_usage BOOLEAN DEFAULT true,
    allow_in_app_rewards BOOLEAN DEFAULT true,
    allow_data_analytics_and_performance BOOLEAN DEFAULT true,
    enable_in_app_notifications BOOLEAN DEFAULT true,
    enable_sound_effects BOOLEAN DEFAULT true,
    enable_vibration BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- City stats table
CREATE TABLE city_stats (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id TEXT UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    state TEXT NOT NULL,
    country TEXT NOT NULL,
    population INTEGER,
    area DECIMAL,
    total_streets_walked INTEGER DEFAULT 0,
    total_kilometers DECIMAL DEFAULT 0,
    city_coverage_pct DECIMAL DEFAULT 0,
    days_active INTEGER DEFAULT 0,
    longest_streak_days INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Street walks table
CREATE TABLE street_walks (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    city_stat_id TEXT NOT NULL REFERENCES city_stats(id) ON DELETE CASCADE,
    street_name TEXT NOT NULL,
    geo_json JSONB NOT NULL,
    distance_km DECIMAL DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Visited streets table
CREATE TABLE visited_streets (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    session_id TEXT NOT NULL,
    street_id TEXT NOT NULL,
    street_name TEXT NOT NULL,
    entry_timestamp BIGINT NOT NULL,
    exit_timestamp BIGINT,
    duration_seconds INTEGER,
    entry_latitude DECIMAL(10,8) NOT NULL,
    entry_longitude DECIMAL(11,8) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Devices table  
CREATE TABLE devices (
    id TEXT PRIMARY KEY DEFAULT gen_random_uuid()::text,
    user_id TEXT UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    location TEXT NOT NULL,
    last_logged_in TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create indexes
CREATE INDEX idx_user_friends_user_id ON user_friends(user_id);
CREATE INDEX idx_user_friends_friend_id ON user_friends(friend_id);
CREATE INDEX idx_visited_streets_user_id ON visited_streets(user_id);
CREATE INDEX idx_visited_streets_session_id ON visited_streets(session_id);
CREATE INDEX idx_visited_streets_street_id ON visited_streets(street_id);
CREATE INDEX idx_visited_streets_entry_timestamp ON visited_streets(entry_timestamp);

-- Update trigger for updated_at columns
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_settings_updated_at BEFORE UPDATE ON settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_city_stats_updated_at BEFORE UPDATE ON city_stats
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();