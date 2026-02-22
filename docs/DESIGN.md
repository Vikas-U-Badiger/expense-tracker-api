# Expense Tracker API - Design Document

## Table of Contents
1. [Overview](#overview)
2. [Architecture](#architecture)
3. [Database Design](#database-design)
4. [Debt Settlement Algorithm](#debt-settlement-algorithm)
5. [Money Handling Strategy](#money-handling-strategy)
6. [Security Considerations](#security-considerations)
7. [API Design Decisions](#api-design-decisions)

## Overview

The Expense Tracker API is a RESTful service designed to help users track shared expenses and split bills among friends. The key challenge addressed is the **debt settlement optimization problem** - minimizing the number of transactions needed to settle all debts within a group.

### Key Features
- User authentication and authorization
- Group management for organizing expenses
- Multiple expense split types (equal, exact, percentage)
- Optimized debt settlement algorithm
- Settlement tracking

## Architecture

### Clean Architecture Pattern

The project follows a layered architecture pattern:

```
┌─────────────────────────────────────┐
│           HTTP Handlers             │
│    (Request/Response handling)      │
├─────────────────────────────────────┤
│           Middleware Layer          │
│  (Auth, CORS, Error Handling, etc)  │
├─────────────────────────────────────┤
│           Service Layer             │
│      (Business Logic)               │
├─────────────────────────────────────┤
│         Repository Layer            │
│    (Database Access)                │
├─────────────────────────────────────┤
│           Models Layer              │
│    (Data Structures)                │
└─────────────────────────────────────┘
```

### Layer Responsibilities

1. **Handlers (Presentation Layer)**
   - Handle HTTP requests and responses
   - Input validation
   - JSON serialization/deserialization
   - No business logic

2. **Middleware**
   - Authentication (JWT verification)
   - CORS handling
   - Error recovery and logging
   - Request context management

3. **Services (Business Logic Layer)**
   - Implement business rules
   - Coordinate between repositories
   - Execute the settlement algorithm
   - Transaction management

4. **Repositories (Data Access Layer)**
   - Database operations
   - Query optimization
   - Data mapping
   - No business logic

5. **Models**
   - Data structures
   - Database entities
   - Request/Response DTOs

## Database Design

### Entity Relationship Diagram

```
┌─────────────┐       ┌─────────────┐       ┌─────────────┐
│    users    │       │   groups    │       │  expenses   │
├─────────────┤       ├─────────────┤       ├─────────────┤
│ id (PK)     │◄──────┤ id (PK)     │◄──────┤ id (PK)     │
│ name        │       │ name        │       │ description │
│ email (UQ)  │       │ description │       │ amount      │
│ password    │       │ created_by  │──────►│ category    │
│ created_at  │       │ created_at  │       │ group_id    │
└─────────────┘       └─────────────┘       │ paid_by     │
       ▲                    ▲               │ expense_date│
       │                    │               │ created_at  │
       │                    │               └─────────────┘
       │                    │                      ▲
       │                    │                      │
       │           ┌─────────────┐          ┌─────────────┐
       │           │group_members│          │expense_shares│
       │           ├─────────────┤          ├─────────────┤
       └───────────┤ user_id(FK) │          │ id (PK)     │
                   │ group_id(FK)│          │ expense_id  │
                   │ joined_at   │          │ user_id     │
                   │ is_active   │          │ amount      │
                   └─────────────┘          └─────────────┘

┌─────────────┐
│ settlements │
├─────────────┤
│ id (PK)     │
│ from_user   │────► users
│ to_user     │────► users
│ group_id    │────► groups (optional)
│ amount      │
│ status      │
│ notes       │
│ settled_at  │
└─────────────┘
```

### Schema Design Decisions

1. **Users Table**
   - UUID primary keys for security and scalability
   - Password hashed with bcrypt
   - Soft delete support (DeletedAt)

2. **Groups Table**
   - Creator tracked for authorization
   - Many-to-many relationship with users via group_members

3. **Expenses Table**
   - Decimal(19,4) for amount to handle large values with precision
   - Category enum for expense classification
   - Expense date separate from created_at for backdating

4. **Expense Shares Table**
   - Separate table for flexible split types
   - Each share linked to a user with specific amount

5. **Settlements Table**
   - Tracks payments between users
   - Optional group link for group-specific settlements
   - Status field for pending/completed/cancelled

### Database Constraints

- Foreign key constraints with CASCADE delete where appropriate
- Unique constraints on email (users)
- Composite primary keys for join tables
- Check constraints on amounts (positive values)

## Debt Settlement Algorithm

### Problem Statement

Given a set of users with net balances (positive = owed to them, negative = they owe), find the minimum number of transactions to settle all debts.

### Algorithm: Greedy Optimization

```
Input: List of (user_id, net_balance) pairs
Output: List of (from_user, to_user, amount) transactions

1. SEPARATE users into:
   - CREDITORS: balance > 0 (owed money)
   - DEBTORS: balance < 0 (owe money)

2. SORT both lists by absolute balance (descending)

3. WHILE creditors not empty AND debtors not empty:
   a. Take largest creditor (C) and largest debtor (D)
   b. settlement = min(C.balance, D.balance)
   c. CREATE transaction: D pays C (settlement amount)
   d. D.balance -= settlement
   e. C.balance -= settlement
   f. IF D.balance ≈ 0: remove D from debtors
   g. IF C.balance ≈ 0: remove C from creditors

4. RETURN all transactions
```

### Example Walkthrough

**Scenario:** 4 users with the following net balances after expenses:
- Alice: +$150 (creditor)
- Bob: -$50 (debtor)
- Charlie: -$50 (debtor)
- David: -$50 (debtor)

**Execution:**
1. Creditors: [Alice: $150]
2. Debtors: [Bob: $50, Charlie: $50, David: $50]
3. Iteration 1: Bob pays Alice $50 → Bob settled
4. Iteration 2: Charlie pays Alice $50 → Charlie settled
5. Iteration 3: David pays Alice $50 → David settled, Alice settled

**Result:** 3 transactions (optimal)

### Complexity Analysis

- **Time Complexity**: O(n log n)
  - Sorting: O(n log n)
  - Matching loop: O(n)
  
- **Space Complexity**: O(n)
  - Storage for creditors and debtors lists

### Why Greedy Works

The greedy approach (matching largest creditor with largest debtor) is optimal for this problem because:

1. **No benefit to partial settlements**: Settling partial amounts doesn't reduce transaction count
2. **Maximizing each transaction**: Largest-first approach maximizes the amount per transaction
3. **Problem structure**: The problem has optimal substructure - each optimal solution contains optimal solutions to subproblems

### Edge Cases Handled

1. **Zero balances**: Users with ~$0 balance are excluded
2. **Single creditor/debtor**: Works correctly
3. **Exact matches**: When creditor = debtor amount, both are settled
4. **Rounding errors**: Small differences (< $0.01) are ignored

## Money Handling Strategy

### Decimal Precision

**Problem**: Floating-point arithmetic (float64) causes precision errors with money.

**Solution**: Use `shopspring/decimal` library

```go
// Storage: 4 decimal places
amount := decimal.NewFromFloat(100.5055).Round(4)
// Result: 100.5055

// Display: 2 decimal places
display := amount.Round(2)
// Result: 100.51
```

### Split Calculations

**Equal Split:**
```go
baseShare := total / numPeople
// Handle remainder by adding to first share
shares[0] += total - (baseShare * numPeople)
```

**Percentage Split:**
```go
share = total * (percentage / 100)
// Validate: sum of percentages = 100
```

**Exact Split:**
```go
// Validate: sum of shares = total amount
valid := math.Abs(total - sum(shares)) < 0.01
```

### Rounding Strategy

1. **Storage**: 4 decimal places for precision
2. **API Response**: 2 decimal places for display
3. **Validation**: Allow ±$0.01 difference due to rounding
4. **Remainder handling**: Add rounding difference to first share

## Security Considerations

### Authentication

- **JWT Tokens**: Stateless authentication
  - Access tokens expire in 24 hours
  - Contains user ID only (no sensitive data)
  - Signed with HS256 algorithm

- **Password Security**:
  - bcrypt hashing with cost 10
  - Minimum password length: 6 characters
  - Passwords never returned in API responses

### Authorization

- **Group-level permissions**:
  - Only members can view group data
  - Only creator can update/delete group
  - Only creator can add/remove members

- **Expense-level permissions**:
  - Only payer can update/delete expense
  - All members can view group expenses

- **Settlement-level permissions**:
  - Only payer can update/cancel settlement
  - Both parties can view

### Input Validation

- Request struct validation using Gin binding tags
- UUID format validation
- Enum value validation (categories, split types)
- Positive amount validation
- SQL injection prevention via GORM

### Data Protection

- Password excluded from all JSON responses (`json:"-"`)
- Soft delete for data recovery
- No sensitive data in logs

## API Design Decisions

### RESTful Principles

1. **Resource-based URLs**: `/users`, `/groups`, `/expenses`
2. **HTTP Methods**: GET, POST, PATCH, DELETE for CRUD
3. **Status Codes**: Appropriate HTTP status codes
   - 200: Success
   - 201: Created
   - 204: No Content (delete)
   - 400: Bad Request
   - 401: Unauthorized
   - 403: Forbidden
   - 404: Not Found

### Pagination

All list endpoints support pagination:
```
GET /api/v1/expenses?page=1&page_size=10
```

Default: page=1, page_size=10
Maximum: page_size=100

### Error Responses

Standard error format:
```json
{
  "error": "Human-readable error message"
}
```

Validation errors include field details:
```json
{
  "error": "Validation failed",
  "errors": ["name is required", "email must be valid"]
}
```

### Response Wrapping

List responses include metadata:
```json
{
  "expenses": [...],
  "total": 100,
  "page": 1,
  "page_size": 10
}
```

### Versioning

API version in URL path:
```
/api/v1/...
```

This allows future API versions without breaking existing clients.

## Performance Considerations

### Database Optimization

1. **Indexes**:
   - Primary keys (UUID)
   - Foreign keys
   - Email (unique)
   - DeletedAt (for soft delete queries)

2. **Eager Loading**:
   - Preload related entities to avoid N+1 queries
   - Example: `Preload("Shares.User")`

3. **Transactions**:
   - Use database transactions for multi-step operations
   - Ensures data consistency

### Caching Opportunities

Future improvements:
- Cache group balances (invalidate on expense/settlement change)
- Cache user summaries
- Redis for session storage (if needed)

## Scalability

### Horizontal Scaling

- Stateless API design (no server-side sessions)
- JWT authentication works across multiple instances
- Database connection pooling

### Database Scaling

- Current: SQLite for simplicity
- Production: PostgreSQL with read replicas
- Connection pooling via GORM

## Testing Strategy

### Unit Tests
- Service layer business logic
- Settlement algorithm
- Money calculations

### Integration Tests
- API endpoint testing
- Database operations
- Authentication flow

### Test Data
- Seed script for test users and groups
- Sample expenses for testing splits

## Future Enhancements

1. **Real-time Updates**: WebSocket for live balance updates
2. **Receipt Upload**: Image storage for expense receipts
3. **Recurring Expenses**: Automatic monthly expense creation
4. **Notifications**: Email/push notifications for settlements
5. **Multi-currency**: Support for different currencies with exchange rates
6. **Expense Categories**: Custom categories per group
7. **Export**: PDF/CSV export of expense reports

## Conclusion

This design document outlines the architecture and key design decisions for the Expense Tracker API. The clean architecture pattern ensures maintainability, while the optimized settlement algorithm provides real value to users by minimizing transaction complexity.
