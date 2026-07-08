BEGIN;

CREATE TABLE IF NOT EXISTS roles (
  id SERIAL PRIMARY KEY,
  name VARCHAR(50) UNIQUE NOT NULL
);

INSERT INTO roles (name) VALUES ('student'), ('teacher'), ('guard'), ('admin')
ON CONFLICT (name) DO NOTHING;

CREATE TABLE IF NOT EXISTS buildings (
  id SERIAL PRIMARY KEY,
  name VARCHAR(100) UNIQUE NOT NULL,
  address VARCHAR(255)
);

CREATE TABLE IF NOT EXISTS users (
  id SERIAL PRIMARY KEY,
  role_id INT NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,
  email VARCHAR(255) UNIQUE NOT NULL,
  last_name VARCHAR(50) NOT NULL,
  first_name VARCHAR(50) NOT NULL,
  patronymic VARCHAR(50),
  phone VARCHAR(16),
  avatar_url TEXT,
  is_active BOOLEAN NOT NULL DEFAULT true,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS groups (
  id SERIAL PRIMARY KEY,
  group_name VARCHAR(50)
);

CREATE TABLE IF NOT EXISTS students (
  student_id INT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  group_id INT NOT NULL REFERENCES groups(id) ON DELETE RESTRICT
);

CREATE TABLE IF NOT EXISTS passwords (
  password_id INT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  password_hash VARCHAR(128) NOT NULL
);

CREATE TABLE IF NOT EXISTS user_devices (
  user_id INT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  device_id VARCHAR(255) UNIQUE NOT NULL,
  secret_key VARCHAR(64) NOT NULL,
  last_used_step BIGINT NULL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS guest_passes (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  last_name VARCHAR(50) NOT NULL,
  first_name VARCHAR(50) NOT NULL,
  patronymic VARCHAR(50),
  purpose TEXT,
  valid_from TIMESTAMP WITH TIME ZONE NOT NULL,
  valid_to TIMESTAMP WITH TIME ZONE NOT NULL,
  is_used BOOLEAN NOT NULL DEFAULT false,
  is_entered BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  CONSTRAINT chk_guest_window CHECK (valid_to > valid_from)
);

CREATE TABLE IF NOT EXISTS access_points (
  id SERIAL PRIMARY KEY,
  building_id INT NOT NULL REFERENCES buildings(id) ON DELETE CASCADE,
  scanner_id VARCHAR(100) UNIQUE NOT NULL,
  gate_number VARCHAR(10) NOT NULL,
  description TEXT
);

CREATE TABLE IF NOT EXISTS access_logs (
  id BIGSERIAL PRIMARY KEY,
  user_id INT REFERENCES users(id) ON DELETE SET NULL,
  guest_pass_id UUID REFERENCES guest_passes(id) ON DELETE SET NULL,
  access_point_id INT NOT NULL REFERENCES access_points(id),
  direction VARCHAR(10) NOT NULL CHECK (direction IN ('enter', 'exit')),
  is_allowed BOOLEAN NOT NULL,
  reason VARCHAR(255),
  logged_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
  CONSTRAINT chk_visitor_type CHECK (
    (user_id IS NOT NULL AND guest_pass_id IS NULL) OR
    (user_id IS NULL AND guest_pass_id IS NOT NULL)
  )
);

CREATE INDEX IF NOT EXISTS idx_access_logs_logged_at ON access_logs (logged_at DESC);
CREATE INDEX IF NOT EXISTS idx_access_logs_user_time ON access_logs (user_id, logged_at DESC);
CREATE INDEX IF NOT EXISTS idx_access_logs_guest_time ON access_logs (guest_pass_id, logged_at DESC);
CREATE INDEX IF NOT EXISTS idx_guest_passes_window ON guest_passes (valid_from, valid_to, is_used);

WITH new_user AS (
INSERT INTO users (
    role_id, email, last_name, first_name, patronymic
)
VALUES (
    (SELECT id FROM roles WHERE name = 'student'),
    'student1@uni.com',
    'test',
    'test',
    'test'
    )
    RETURNING id
    )
INSERT INTO passwords (password_id, password_hash)
SELECT id, '$2a$10$N.VTcjd1Yw9dYqcPNpGuSO0RqASum/Jfp6ktBJIafn2VEMoYuT5ve'
FROM new_user;

INSERT INTO guest_passes (
    id,
    last_name,
    first_name,
    patronymic,
    purpose,
    valid_from,
    valid_to,
    is_used,
    is_entered
)
VALUES (
           '550e8400-e29b-41d4-a716-446655440000',
           'guest',
           'guest',
           'guest',
           'Event guest',
           NOW() - INTERVAL '5 minutes',
           NOW() + INTERVAL '30 minutes',
           FALSE,
           FALSE
       );

INSERT INTO buildings (name, address)
SELECT 'Main', 'Bolshaya Morskaya'
WHERE NOT EXISTS (SELECT 1 FROM buildings WHERE name = 'Main Building');

-- тестовая точка
INSERT INTO access_points (building_id, scanner_id, gate_number, description)
SELECT b.id, 'SCANNER_001', 'G1', 'Main entrance'
FROM buildings b
WHERE b.name = 'Main'
  AND NOT EXISTS (SELECT 1 FROM access_points WHERE scanner_id = 'SCANNER_001');

COMMIT;
