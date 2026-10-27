-- ==============================================================================
-- PostgreSQL Production Initialization Script
-- ==============================================================================

-- Ensure the application database exists
SELECT 'CREATE DATABASE gestion_immo ENCODING ''UTF8'''
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'gestion_immo')\gexec

-- Configure production defaults on gestion_immo
\connect gestion_immo
ALTER DATABASE gestion_immo SET timezone TO 'UTC';
