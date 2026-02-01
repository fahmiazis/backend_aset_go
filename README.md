# Backend Go - RESTful API with Gin & GORM

A production-ready RESTful API built with Go, featuring JWT authentication, role-based access control, and MySQL database integration.

## 🚀 Features

- ✅ RESTful API with Gin framework
- ✅ JWT-based authentication (Access & Refresh tokens)
- ✅ Role-based authorization (RBAC)
- ✅ MySQL database with GORM ORM
- ✅ Database migrations with Goose
- ✅ Docker & Docker Compose support
- ✅ Makefile for common tasks
- ✅ CORS enabled
- ✅ Password hashing with bcrypt
- ✅ UUID for primary keys
- ✅ Soft deletes
- ✅ Multi-device session management
- ✅ Clean architecture (MVC pattern)

## 📁 Project Structure
```
backend-go/
├── config/              # Database configuration
├── controllers/         # Request handlers
├── dto/                 # Data Transfer Objects
├── middleware/          # Auth & authorization middleware
├── migrations/          # Database migrations (Goose)
├── models/              # Database models
├── routes/              # API routes
├── services/            # Business logic
├── utils/               # Helper functions (JWT, response)
├── .env                 # Environment variables
├── .env.example         # Environment template
├── .dockerignore        # Docker ignore file
├── .gitignore           # Git ignore file
├── docker-compose.yml   # Docker Compose configuration
├── Dockerfile           # Docker image definition
├── go.mod               # Go module dependencies
├── go.sum               # Dependency checksums
├── main.go              # Application entry point
├── Makefile             # Build automation
└── README.md            # This file
```

## 🛠️ Tech Stack

- **Language:** Go 1.25+
- **Web Framework:** Gin
- **ORM:** GORM
- **Database:** MySQL 8.0
- **Migration Tool:** Goose
- **Authentication:** JWT (golang-jwt/jwt/v5)
- **Password Hashing:** bcrypt
- **Containerization:** Docker & Docker Compose

## 📋 Prerequisites

### Local Development
- Go 1.25 or higher
- MySQL 8.0 or higher
- Goose CLI (for migrations)

### Docker (Recommended)
- Docker
- Docker Compose

## 🚀 Quick Start

### 1. Clone the repository
```bash
git clone <repository-url>
cd backend-go
```

### 2. Setup environment variables
```bash
cp .env.example .env
```

Edit `.env` with your configuration:
```env
# Database
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=yourpassword
DB_NAME=go_rest_api

# Application
PORT=8080

# JWT
JWT_SECRET=your-super-secret-key-change-this-in-production-min-32-chars
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=168h
```

### 3. Run with Docker (Recommended)
```bash
# Build and start all services
make deploy

# Or step by step:
make docker-build      # Build Docker image
make docker-up         # Start services
make docker-migrate-up # Run migrations

# View logs
make docker-logs

# Stop services
make docker-down
```

### 4. Run Locally (Without Docker)
```bash
# Install dependencies
make deps

# Install Goose (if not installed)
go install github.com/pressly/goose/v3/cmd/goose@latest

# Create database
mysql -u root -p
CREATE DATABASE go_rest_api;

# Run migrations
make migrate-up

# Run application
make run
```

Application will be available at `http://localhost:8080`

## 📚 API Documentation

### Base URL
```
http://localhost:8080/api/v1
```

### Health Check
```
GET /health
```

### Authentication Endpoints

#### Register
```http
POST /api/v1/auth/register
Content-Type: application/json

{
  "username": "john_doe",
  "fullname": "John Doe",
  "email": "john@example.com",
  "password": "password123",
  "nik": "1234567890",
  "mpn_number": "MPN001"
}
```

#### Login
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "john_doe",
  "password": "password123"
}

Response:
{
  "status": "success",
  "message": "Login successful",
  "data": {
    "access_token": "eyJhbGci...",
    "refresh_token": "eyJhbGci...",
    "token_type": "Bearer",
    "expires_in": 900,
    "user": {
      "id": "uuid",
      "username": "john_doe",
      "email": "john@example.com",
      "roles": ["user"]
    }
  }
}
```

#### Refresh Token
```http
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGci..."
}
```

#### Get Current User Profile
```http
GET /api/v1/auth/me
Authorization: Bearer {access_token}
```

#### Logout
```http
POST /api/v1/auth/logout
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "refresh_token": "eyJhbGci..."
}
```

#### Logout All Devices
```http
POST /api/v1/auth/logout-all
Authorization: Bearer {access_token}
```

### User Management Endpoints (Admin Only)

#### Get All Users
```http
GET /api/v1/users
Authorization: Bearer {access_token}
```

#### Get User by ID
```http
GET /api/v1/users/{id}
Authorization: Bearer {access_token}
```

#### Create User
```http
POST /api/v1/users
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "username": "jane_doe",
  "fullname": "Jane Doe",
  "email": "jane@example.com",
  "password": "password123",
  "status": "active"
}
```

#### Update User
```http
PUT /api/v1/users/{id}
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "fullname": "Jane Smith",
  "email": "jane.smith@example.com"
}
```

#### Delete User
```http
DELETE /api/v1/users/{id}
Authorization: Bearer {access_token}
```

#### Assign Roles
```http
POST /api/v1/users/{id}/roles
Authorization: Bearer {access_token}
Content-Type: application/json

{
  "role_ids": ["role-uuid-1", "role-uuid-2"]
}
```

## 🔐 Default Roles

The system comes with three default roles:

- **admin**: Full access to all resources
- **manager**: Elevated permissions (read, write, approve)
- **user**: Basic read access

## 🗄️ Database Schema

### Users Table
- `id` (UUID, PK)
- `username` (unique)
- `fullname`
- `email` (unique)
- `password` (bcrypt hashed)
- `nik` (unique, nullable)
- `mpn_number` (nullable)
- `status` (active/inactive)
- `created_at`, `updated_at`, `deleted_at`

### Refresh Tokens Table
- `id` (UUID, PK)
- `user_id` (FK to users)
- `token`
- `device_info`
- `ip_address`
- `expires_at`
- `is_revoked`
- `created_at`

### Roles Table
- `id` (UUID, PK)
- `name` (unique)
- `description`
- `permissions` (JSON)
- `created_at`, `updated_at`

### User Roles Table (Many-to-Many)
- `id` (UUID, PK)
- `user_id` (FK to users)
- `role_id` (FK to roles)
- `created_at`

## 🔧 Makefile Commands

### Development
```bash
make build          # Build the application
make run            # Run the application
make test           # Run tests
make clean          # Clean build artifacts
make deps           # Install dependencies
make dev            # Start dev environment (MySQL in Docker, app local)
```

### Migrations
```bash
make migrate-up                    # Run all migrations
make migrate-down                  # Rollback last migration
make migrate-status                # Check migration status
make migrate-create NAME=<name>    # Create new migration
```

### Docker
```bash
make docker-build      # Build Docker image
make docker-up         # Start all services
make docker-down       # Stop all services
make docker-logs       # View logs
make docker-restart    # Restart services
make docker-clean      # Remove containers and volumes
make deploy            # Full deployment (build + migrate + run)
```

### Docker Migrations
```bash
make docker-migrate-up       # Run migrations in Docker
make docker-migrate-down     # Rollback migration in Docker
make docker-migrate-status   # Check migration status in Docker
```

## 🧪 Testing
```bash
# Run all tests
make test

# Test API with curl
curl http://localhost:8080/health

# Expected response:
{
  "status": "ok",
  "message": "Server is running"
}
```

## 📝 Migration Workflow

### Create a new migration
```bash
make migrate-create NAME=add_user_phone
```

This creates two files in `migrations/`:
- `XXXXXX_add_user_phone.sql`

Edit the file:
```sql
-- +goose Up
ALTER TABLE users ADD COLUMN phone VARCHAR(20);

-- +goose Down
ALTER TABLE users DROP COLUMN phone;
```

### Apply migrations
```bash
# Local
make migrate-up

# Docker
make docker-migrate-up
```

### Rollback migration
```bash
# Local
make migrate-down

# Docker
make docker-migrate-down
```

## 🔒 Security Best Practices

- ✅ Passwords are hashed with bcrypt
- ✅ JWT tokens with configurable expiry
- ✅ Refresh token rotation
- ✅ Multi-device session management
- ✅ Role-based access control
- ✅ CORS enabled with whitelist
- ✅ SQL injection prevention (GORM parameterized queries)
- ✅ Non-root user in Docker container

## 🌍 Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_HOST` | Database host | `localhost` |
| `DB_PORT` | Database port | `3306` |
| `DB_USER` | Database user | `root` |
| `DB_PASSWORD` | Database password | - |
| `DB_NAME` | Database name | `go_rest_api` |
| `PORT` | Application port | `8080` |
| `JWT_SECRET` | JWT signing secret (min 32 chars) | - |
| `JWT_ACCESS_EXPIRY` | Access token expiry | `15m` |
| `JWT_REFRESH_EXPIRY` | Refresh token expiry | `168h` (7 days) |

## 🐛 Troubleshooting

### Port already in use
```bash
# Find process using port 8080
lsof -i :8080

# Kill the process
kill -9 <PID>
```

### Database connection error
- Check MySQL is running
- Verify credentials in `.env`
- Ensure database exists: `CREATE DATABASE go_rest_api;`

### Migration errors
```bash
# Check migration status
make migrate-status

# Reset database (WARNING: deletes all data)
make docker-down
make docker-clean
make deploy
```

### Docker build fails
```bash
# Clean Docker cache
docker system prune -a
make docker-build
```

## 📦 Deployment

### Production Environment

1. Update `.env` with production values
2. Change `JWT_SECRET` to a strong random string
3. Update `AllowOrigins` in CORS config
4. Use environment-specific `.env` files
5. Enable HTTPS/TLS
```bash
# Deploy to production
make deploy
```

### Docker Hub (Optional)
```bash
# Build and tag
docker build -t username/backend-go:latest .

# Push to Docker Hub
docker push username/backend-go:latest

# Pull and run on server
docker pull username/backend-go:latest
docker-compose up -d
```