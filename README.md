# Expense Tracker API

A production-ready REST API for tracking shared expenses and splitting bills among friends, similar to Splitwise. Built with Go (Golang), Gin framework, and GORM.

## Features

- **User Management**: Registration, authentication with JWT, profile management
- **Group Management**: Create groups, add/remove members
- **Expense Tracking**: Create expenses with multiple split types (equal, exact, percentage)
- **Smart Settlement Algorithm**: Optimizes debt settlement to minimize number of transactions
- **Balance Management**: Real-time balance calculation across groups
- **Settlement Recording**: Track payments between users

## Tech Stack

- **Language**: Go 1.21+
- **Web Framework**: Gin
- **ORM**: GORM
- **Database**: SQLite (default) / PostgreSQL
- **Authentication**: JWT (JSON Web Tokens)
- **Password Hashing**: bcrypt
- **Decimal Precision**: shopspring/decimal for money handling

## Project Structure

```
expense-tracker-api/
├── cmd/
│   └── main.go                 # Application entry point
├── internal/
│   ├── config/                 # Configuration and database setup
│   ├── handlers/               # HTTP request handlers
│   ├── middleware/             # Authentication, CORS, error handling
│   ├── models/                 # Data models and request/response types
│   ├── repositories/           # Database access layer
│   └── services/               # Business logic layer
├── pkg/
│   └── settlement/             # Debt settlement algorithm
├── docs/                       # Documentation
├── postman/                    # Postman collection
├── migrations/                 # Database migrations
├── go.mod
├── go.sum
└── README.md
```

## Quick Start

### Prerequisites

- Go 1.21 or higher
- (Optional) PostgreSQL database

### Installation

1. Clone the repository:
```bash
git clone <repository-url>
cd expense-tracker-api
```

2. Install dependencies:
```bash
go mod download
```

3. Run the application:
```bash
go run cmd/main.go
```

The server will start on `http://localhost:8080`

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `SERVER_PORT` | Server port | `8080` |
| `ENVIRONMENT` | Environment (development/production) | `development` |
| `DB_DRIVER` | Database driver (sqlite/postgres) | `sqlite` |
| `DB_HOST` | Database host (PostgreSQL) | `localhost` |
| `DB_PORT` | Database port (PostgreSQL) | `5432` |
| `DB_USER` | Database user (PostgreSQL) | `postgres` |
| `DB_PASSWORD` | Database password (PostgreSQL) | `password` |
| `DB_NAME` | Database name (PostgreSQL) | `expense_tracker` |
| `JWT_SECRET` | JWT signing secret | `your-super-secret-jwt-key-change-in-production` |
| `JWT_TOKEN_EXPIRY` | JWT token expiry in hours | `24` |

## API Documentation

### Authentication

All protected endpoints require a Bearer token in the Authorization header:
```
Authorization: Bearer <your-jwt-token>
```

### Endpoints

#### Users

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| POST | `/api/v1/users/register` | Register a new user | No |
| POST | `/api/v1/users/login` | Login and get token | No |
| GET | `/api/v1/users/me` | Get current user profile | Yes |
| PATCH | `/api/v1/users/me` | Update current user profile | Yes |
| GET | `/api/v1/users/dashboard` | Get user dashboard data | Yes |
| GET | `/api/v1/users` | List all users | Yes |
| GET | `/api/v1/users/:id` | Get user by ID | Yes |

**Register Request:**
```json
{
  "name": "John Doe",
  "email": "john@example.com",
  "password": "password123"
}
```

**Login Request:**
```json
{
  "email": "john@example.com",
  "password": "password123"
}
```

**Login Response:**
```json
{
  "user": {
    "id": "uuid",
    "name": "John Doe",
    "email": "john@example.com",
    "created_at": "2024-01-01T00:00:00Z"
  },
  "token": "eyJhbGciOiJIUzI1NiIs..."
}
```

#### Groups

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| POST | `/api/v1/groups` | Create a new group | Yes |
| GET | `/api/v1/groups` | List user's groups | Yes |
| GET | `/api/v1/groups/:id` | Get group details with balances | Yes |
| PATCH | `/api/v1/groups/:id` | Update group (creator only) | Yes |
| DELETE | `/api/v1/groups/:id` | Delete group (creator only) | Yes |
| POST | `/api/v1/groups/:id/members` | Add member to group | Yes |
| DELETE | `/api/v1/groups/:id/members/:user_id` | Remove member from group | Yes |
| GET | `/api/v1/groups/:id/balances` | Get group balances | Yes |
| GET | `/api/v1/groups/:id/simplified-debts` | Get optimized debts | Yes |

**Create Group Request:**
```json
{
  "name": "Trip to Goa",
  "description": "Weekend trip expenses",
  "member_ids": ["user-uuid-1", "user-uuid-2"]
}
```

**Group Detail Response:**
```json
{
  "id": "group-uuid",
  "name": "Trip to Goa",
  "description": "Weekend trip expenses",
  "created_by_id": "user-uuid",
  "member_count": 3,
  "members": [...],
  "balances": [
    {
      "user_id": "user-uuid-1",
      "user_name": "Alice",
      "balance": "150.00"
    },
    {
      "user_id": "user-uuid-2",
      "user_name": "Bob",
      "balance": "-150.00"
    }
  ],
  "simplified_debts": [
    {
      "from_user_id": "user-uuid-2",
      "from_user_name": "Bob",
      "to_user_id": "user-uuid-1",
      "to_user_name": "Alice",
      "amount": "150.00"
    }
  ],
  "total_expenses": "450.00",
  "created_at": "2024-01-01T00:00:00Z"
}
```

#### Expenses

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| POST | `/api/v1/expenses` | Create a new expense | Yes |
| GET | `/api/v1/expenses` | List user's expenses | Yes |
| GET | `/api/v1/expenses/summary` | Get expense summary | Yes |
| GET | `/api/v1/expenses/group/:group_id` | List group expenses | Yes |
| GET | `/api/v1/expenses/:id` | Get expense details | Yes |
| PATCH | `/api/v1/expenses/:id` | Update expense (payer only) | Yes |
| DELETE | `/api/v1/expenses/:id` | Delete expense (payer only) | Yes |

**Create Expense Request (Equal Split):**
```json
{
  "description": "Dinner at Restaurant",
  "amount": "120.00",
  "category": "food",
  "group_id": "group-uuid",
  "expense_date": "2024-01-15T19:00:00Z",
  "split_type": "equal",
  "shares": [
    {"user_id": "user-uuid-1"},
    {"user_id": "user-uuid-2"},
    {"user_id": "user-uuid-3"}
  ]
}
```

**Create Expense Request (Exact Split):**
```json
{
  "description": "Hotel Booking",
  "amount": "300.00",
  "category": "housing",
  "group_id": "group-uuid",
  "split_type": "exact",
  "shares": [
    {"user_id": "user-uuid-1", "amount": "100.00"},
    {"user_id": "user-uuid-2", "amount": "100.00"},
    {"user_id": "user-uuid-3", "amount": "100.00"}
  ]
}
```

**Create Expense Request (Percentage Split):**
```json
{
  "description": "Groceries",
  "amount": "200.00",
  "category": "food",
  "group_id": "group-uuid",
  "split_type": "percent",
  "shares": [
    {"user_id": "user-uuid-1", "amount": "50"},
    {"user_id": "user-uuid-2", "amount": "30"},
    {"user_id": "user-uuid-3", "amount": "20"}
  ]
}
```

**Expense Categories:**
- `food`
- `transport`
- `housing`
- `utilities`
- `entertainment`
- `shopping`
- `health`
- `education`
- `travel`
- `other`

#### Settlements

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|---------------|
| POST | `/api/v1/settlements` | Record a settlement | Yes |
| GET | `/api/v1/settlements` | List user's settlements | Yes |
| GET | `/api/v1/settlements/summary` | Get settlement summary | Yes |
| GET | `/api/v1/settlements/group/:group_id` | List group settlements | Yes |
| GET | `/api/v1/settlements/balance/:user_id` | Get balance with user | Yes |
| GET | `/api/v1/settlements/:id` | Get settlement details | Yes |
| PATCH | `/api/v1/settlements/:id` | Update settlement notes | Yes |
| DELETE | `/api/v1/settlements/:id` | Cancel settlement | Yes |

**Create Settlement Request:**
```json
{
  "to_user_id": "user-uuid",
  "group_id": "group-uuid",
  "amount": "50.00",
  "notes": "Payment for dinner",
  "settled_at": "2024-01-20T10:00:00Z"
}
```

## Debt Settlement Algorithm

The API includes an optimized debt settlement algorithm that minimizes the number of transactions needed to settle all debts within a group.

### How It Works

1. **Calculate Net Balances**: For each user, calculate what they're owed minus what they owe
2. **Separate Creditors and Debtors**: Users with positive balance are creditors, negative are debtors
3. **Greedy Matching**: Match the largest debtor with the largest creditor
4. **Optimize**: Continue until all debts are settled

### Example

**Before Optimization:**
- Alice paid $300, Bob owes Alice $100, Charlie owes Alice $100, David owes Alice $100
- **Transactions needed: 3** (Bob→Alice, Charlie→Alice, David→Alice)

**After Optimization (same scenario):**
- The algorithm keeps it at 3 transactions (already optimal)

**Complex Example:**
- Alice: +$150 (owed)
- Bob: -$50 (owes)
- Charlie: -$50 (owes)
- David: -$50 (owes)

**Result:** 3 transactions (Bob→Alice $50, Charlie→Alice $50, David→Alice $50)

### Algorithm Complexity

- **Time Complexity**: O(n log n) due to sorting
- **Space Complexity**: O(n)

## Testing with cURL

### Register a User
```bash
curl -X POST http://localhost:8080/api/v1/users/register \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Alice",
    "email": "alice@example.com",
    "password": "password123"
  }'
```

### Login
```bash
curl -X POST http://localhost:8080/api/v1/users/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "password": "password123"
  }'
```

### Create a Group
```bash
curl -X POST http://localhost:8080/api/v1/groups \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "name": "Roommates",
    "description": "Monthly expenses"
  }'
```

### Create an Expense
```bash
curl -X POST http://localhost:8080/api/v1/expenses \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "description": "Groceries",
    "amount": "150.00",
    "category": "food",
    "group_id": "<group-uuid>",
    "split_type": "equal",
    "shares": [
      {"user_id": "<user1-uuid>"},
      {"user_id": "<user2-uuid>"}
    ]
  }'
```

### Get Simplified Debts
```bash
curl http://localhost:8080/api/v1/groups/<group-uuid>/simplified-debts \
  -H "Authorization: Bearer <token>"
```

## Money Handling

This API uses `shopspring/decimal` for all monetary calculations to avoid floating-point precision issues.

- All amounts are stored with 4 decimal places precision
- API responses round to 2 decimal places for display
- Validation ensures amounts are positive
- Split calculations handle rounding differences automatically

## Security Features

- **Password Hashing**: bcrypt with default cost
- **JWT Authentication**: Stateless token-based auth
- **Input Validation**: Request validation using Gin binding
- **SQL Injection Protection**: GORM parameterized queries
- **CORS**: Configurable cross-origin resource sharing

## Development

### Running Tests
```bash
go test ./...
```

### Building for Production
```bash
go build -o expense-tracker-api cmd/main.go
```

### Using PostgreSQL

1. Set environment variables:
```bash
export DB_DRIVER=postgres
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=postgres
export DB_PASSWORD=yourpassword
export DB_NAME=expense_tracker
```

2. Run the application


