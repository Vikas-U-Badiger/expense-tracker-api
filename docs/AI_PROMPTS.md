# AI Prompts Used for Development

This document contains all the AI prompts used during the development of the Expense Tracker API for transparency, as required by the Infosys project guidelines.

## Overview

The development of this project involved using AI assistance for code generation, architecture design, and documentation. All prompts used are documented below.

---

## Prompt 1: Project Structure and Architecture

**Prompt:**
```
I need to build a Go REST API for an expense tracker with bill splitting functionality similar to Splitwise. 

Requirements:
1. Clean architecture with handlers, services, repositories pattern
2. JWT authentication
3. Database using GORM with SQLite/PostgreSQL support
4. Proper money handling with decimal precision
5. A debt settlement algorithm to minimize transactions

Please provide:
1. Complete project structure
2. Database schema design
3. Core models (User, Group, Expense, Settlement)
4. The settlement algorithm implementation
```

**Response Summary:**
AI provided the layered architecture pattern with:
- Handler layer for HTTP requests
- Service layer for business logic
- Repository layer for database access
- Models for data structures
- Settlement algorithm using greedy approach

---

## Prompt 2: Database Models Design

**Prompt:**
```
Design Go structs for an expense tracker with these requirements:

1. User: id, name, email, password, timestamps
2. Group: id, name, description, creator, members (many-to-many)
3. Expense: id, description, amount, category, group, paid_by, shares
4. Settlement: id, from_user, to_user, amount, status

Use:
- UUID for IDs
- shopspring/decimal for money
- GORM tags for database mapping
- JSON tags for API responses
```

**Response Summary:**
AI generated complete model structs with:
- Proper GORM relationships
- Request/Response DTOs
- Validation tags
- Helper methods for conversions

---

## Prompt 3: Debt Settlement Algorithm

**Prompt:**
```
Implement a debt settlement algorithm in Go that minimizes the number of transactions needed to settle all debts in a group.

Input: List of users with net balances (positive = owed to them, negative = they owe)
Output: List of transactions (from, to, amount)

Requirements:
1. Use greedy approach for optimization
2. Handle edge cases (zero balances, exact matches)
3. Use decimal for precision
4. Time complexity should be O(n log n)

Example:
- Alice: +$150
- Bob: -$50
- Charlie: -$50
- David: -$50

Should return 3 transactions.
```

**Response Summary:**
AI provided the greedy algorithm implementation:
1. Separate creditors and debtors
2. Sort by absolute amount (descending)
3. Match largest debtor with largest creditor
4. Continue until all settled

---

## Prompt 4: JWT Authentication Middleware

**Prompt:**
```
Create JWT authentication middleware for Gin framework in Go with:

1. Token generation with user_id claim
2. Middleware to verify tokens from Authorization header
3. Extract user_id and store in context
4. Helper function to get user_id from context
5. Proper error handling

Use golang-jwt/jwt/v5 library.
```

**Response Summary:**
AI generated complete middleware with:
- Token generation with expiry
- Bearer token extraction
- JWT verification
- Context management
- Error responses

---

## Prompt 5: API Handler Structure

**Prompt:**
```
Create Gin HTTP handlers for a REST API with these endpoints:

Users:
- POST /users/register
- POST /users/login
- GET /users/me

Groups:
- POST /groups
- GET /groups/:id
- GET /groups/:id/balances
- GET /groups/:id/simplified-debts

Expenses:
- POST /expenses
- GET /expenses/:id
- DELETE /expenses/:id

Include:
1. Input validation
2. Proper HTTP status codes
3. JSON request/response handling
4. Authentication middleware integration
```

**Response Summary:**
AI provided handler structure with:
- Request binding and validation
- Service layer calls
- JSON responses
- Error handling
- Route registration methods

---

## Prompt 6: Money Handling with Decimal

**Prompt:**
```
Explain best practices for handling money in Go and provide examples:

1. Why not use float64 for money?
2. How to use shopspring/decimal?
3. How to handle rounding in split calculations?
4. Database storage for decimal values
5. Validation for money amounts
```

**Response Summary:**
AI explained:
- Floating-point precision issues
- Decimal library usage
- Rounding strategies (banker's rounding)
- Database DECIMAL(19,4) type
- Positive amount validation

---

## Prompt 7: Database Configuration

**Prompt:**
```
Create database configuration in Go that supports both SQLite and PostgreSQL:

Requirements:
1. Environment-based configuration
2. GORM initialization
3. Auto-migration for models
4. Connection string generation
5. Default to SQLite for development
```

**Response Summary:**
AI provided:
- Config struct with database settings
- SQLite and PostgreSQL dialector selection
- Database URL generation
- Auto-migrate function
- Environment variable support

---

## Prompt 8: README Documentation

**Prompt:**
```
Write a comprehensive README.md for a Go REST API project with:

1. Project overview and features
2. Tech stack
3. Project structure
4. Installation instructions
5. API documentation with example requests/responses
6. Environment variables
7. Testing with curl examples
8. Algorithm explanation
```

**Response Summary:**
AI generated complete README with:
- Feature list
- Tech stack table
- Directory structure
- Setup instructions
- API endpoint table
- curl examples
- Algorithm description

---

## Prompt 9: Design Document

**Prompt:**
```
Create a design document for an expense tracker API covering:

1. Architecture pattern (Clean Architecture)
2. Database schema with ER diagram
3. Debt settlement algorithm with pseudocode
4. Money handling strategy
5. Security considerations
6. API design decisions
7. Performance considerations
```

**Response Summary:**
AI provided comprehensive design doc with:
- Layered architecture explanation
- Entity relationship diagram
- Algorithm pseudocode and complexity analysis
- Money handling best practices
- Security features
- API design rationale

---

## Prompt 10: Postman Collection

**Prompt:**
```
Create a Postman collection JSON for testing these API endpoints:

Users:
- Register, Login, Get Profile

Groups:
- Create, Get, Update, Delete
- Add/Remove members
- Get balances, Get simplified debts

Expenses:
- Create (equal, exact, percentage splits)
- List, Get, Update, Delete

Settlements:
- Create, List, Get, Cancel
- Get balance with user

Include:
- Variables for base_url, token, group_id
- Example request bodies
- Descriptions for each endpoint
```

**Response Summary:**
AI generated complete Postman collection with:
- All endpoints organized by category
- Request examples with proper bodies
- Variable placeholders
- Authentication headers
- Descriptions

---

## Summary of AI Usage

### What AI Helped With:
1. **Architecture Design**: Clean architecture pattern, layer separation
2. **Code Generation**: Boilerplate code, struct definitions, middleware
3. **Algorithm Implementation**: Debt settlement algorithm logic
4. **Documentation**: README, design document, API docs
5. **Best Practices**: Money handling, security, error handling

### What Was Manually Implemented/Modified:
1. **Business Logic**: Service layer implementations
2. **Repository Queries**: Complex database queries
3. **Edge Cases**: Error handling, validation rules
4. **Integration**: Connecting all layers together
5. **Testing**: Manual testing and verification

### AI Tools Used:
- ChatGPT/GPT-4 for code generation and documentation

### Transparency Note:
All code generated by AI was reviewed, tested, and modified as needed to ensure correctness and meet project requirements. The settlement algorithm was verified against multiple test cases.

---

## Verification

The following components were manually verified:

1. ✅ All API endpoints return correct responses
2. ✅ JWT authentication works correctly
3. ✅ Database relationships are properly configured
4. ✅ Settlement algorithm produces optimal results
5. ✅ Money calculations are precise
6. ✅ Error handling covers edge cases
7. ✅ Documentation is accurate and complete

---

## License

This project was created as part of the Infosys Capstone Project. All AI-generated code is used in compliance with academic integrity guidelines.
