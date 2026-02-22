# Quick Start Guide

## Prerequisites

- Go 1.21 or higher installed
- (Optional) Docker and Docker Compose
- (Optional) PostgreSQL if not using SQLite

## Option 1: Run with Go (Recommended for Development)

### Step 1: Clone and Navigate
```bash
cd expense-tracker-api
```

### Step 2: Install Dependencies
```bash
go mod download
```

### Step 3: Run the Application
```bash
go run cmd/main.go
```

The server will start on `http://localhost:8080`

### Step 4: Test the API
```bash
# Health check
curl http://localhost:8080/health

# Register a user
curl -X POST http://localhost:8080/api/v1/users/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Alice",
    "email": "alice@example.com",
    "password": "password123"
  }'

# Login
curl -X POST http://localhost:8080/api/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "password": "password123"
  }'
```

## Option 2: Run with Docker

### Step 1: Build and Run
```bash
docker-compose up --build
```

This will start:
- API server on port 8080
- PostgreSQL database on port 5432

### Step 2: Stop Services
```bash
docker-compose down
```

To remove data volume:
```bash
docker-compose down -v
```

## Option 3: Run with Makefile

### Available Commands
```bash
# Build the application
make build

# Run the application
make run

# Run with hot reload (requires air)
make watch

# Run tests
make test

# Clean build artifacts and database
make clean

# Show all available commands
make help
```

## Configuration

### Environment Variables

Copy the example environment file:
```bash
cp .env.example .env
```

Edit `.env` to customize settings:

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_PORT` | Server port | `8080` |
| `ENVIRONMENT` | Environment mode | `development` |
| `DB_DRIVER` | Database (sqlite/postgres) | `sqlite` |
| `JWT_SECRET` | JWT signing key | (generate new) |

### Using PostgreSQL

1. Set `DB_DRIVER=postgres` in `.env`
2. Configure PostgreSQL connection details
3. Run the application

## API Testing with Postman

1. Import the collection from `postman/Expense-Tracker-API.postman_collection.json`
2. Set the `base_url` variable to `http://localhost:8080`
3. Run the "Register User" request
4. Copy the token from the response
5. Set the `token` variable in Postman
6. Test other endpoints

## Common Workflows

### 1. Create a Group and Add Expense

```bash
# 1. Register and login to get token
TOKEN=$(curl -s -X POST http://localhost:8080/api/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{"email":"alice@example.com","password":"password123"}' | jq -r '.token')

# 2. Create a group
curl -X POST http://localhost:8080/api/v1/groups \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name":"Roommates","description":"Monthly expenses"}'

# 3. Create an expense (replace GROUP_ID with actual ID)
curl -X POST http://localhost:8080/api/v1/expenses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "description": "Groceries",
    "amount": "150.00",
    "category": "food",
    "group_id": "GROUP_ID",
    "split_type": "equal",
    "shares": [
      {"user_id": "USER_ID_1"},
      {"user_id": "USER_ID_2"}
    ]
  }'

# 4. Get simplified debts
curl http://localhost:8080/api/v1/groups/GROUP_ID/simplified-debts \
  -H "Authorization: Bearer $TOKEN"
```

### 2. Record a Settlement

```bash
# Record a payment from current user to another user
curl -X POST http://localhost:8080/api/v1/settlements \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{
    "to_user_id": "OTHER_USER_ID",
    "group_id": "GROUP_ID",
    "amount": "50.00",
    "notes": "Payment for groceries"
  }'
```

## Troubleshooting

### Port Already in Use
```bash
# Find process using port 8080
lsof -i :8080

# Kill the process
kill -9 <PID>
```

### Database Issues
```bash
# Clean SQLite database
rm expense_tracker.db*

# Or use make
clean
```

### Permission Denied (Docker)
```bash
# Run with sudo or add user to docker group
sudo usermod -aG docker $USER
# Logout and login again
```

## Next Steps

1. Read the [README.md](README.md) for detailed API documentation
2. Check [DESIGN.md](docs/DESIGN.md) for architecture details
3. Review [AI_PROMPTS.md](docs/AI_PROMPTS.md) for transparency
4. Import the Postman collection for testing

## Support

For issues or questions, refer to:
- API Documentation: README.md
- Design Document: docs/DESIGN.md
- Example Requests: postman/Expense-Tracker-API.postman_collection.json
