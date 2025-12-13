-- PostgreSQL initialization script
-- This script sets up both the main odata database and keycloak database with their users

-- Create users
CREATE USER odata WITH PASSWORD 'odata_dev';
CREATE USER keycloak WITH PASSWORD 'keycloak_dev';

-- Create databases
CREATE DATABASE odata_test OWNER odata;
CREATE DATABASE keycloak OWNER keycloak;

-- Grant privileges on odata_test database
GRANT ALL PRIVILEGES ON DATABASE odata_test TO odata;

-- Grant privileges on keycloak database
GRANT ALL PRIVILEGES ON DATABASE keycloak TO keycloak;

-- Connect to odata_test database and grant schema privileges
\c odata_test;
GRANT ALL ON SCHEMA public TO odata;
ALTER DEFAULT PRIVILEGES FOR USER postgres IN SCHEMA public GRANT ALL ON TABLES TO odata;
ALTER DEFAULT PRIVILEGES FOR USER postgres IN SCHEMA public GRANT ALL ON SEQUENCES TO odata;

-- Connect to keycloak database and grant schema privileges
\c keycloak;
GRANT ALL ON SCHEMA public TO keycloak;
ALTER DEFAULT PRIVILEGES FOR USER postgres IN SCHEMA public GRANT ALL ON TABLES TO keycloak;
ALTER DEFAULT PRIVILEGES FOR USER postgres IN SCHEMA public GRANT ALL ON SEQUENCES TO keycloak;
