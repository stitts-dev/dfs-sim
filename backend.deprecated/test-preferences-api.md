# User Preferences API Testing Guide

## Overview
The User Preferences API provides endpoints to manage user interface preferences, tooltip settings, and personalization options.

## Authentication
All preferences endpoints require JWT authentication. Include the bearer token in the Authorization header:
```
Authorization: Bearer <your-jwt-token>
```

## Endpoints

### 1. Get User Preferences
**GET** `/api/v1/user/preferences`

Returns the current user's preferences. If no preferences exist, they will be created with default values.

**Example Response:**
```json
{
  "success": true,
  "data": {
    "user_id": 123,
    "beginner_mode": false,
    "show_tooltips": true,
    "tooltip_delay": 500,
    "preferred_sports": ["nfl", "nba"],
    "ai_suggestions_enabled": true,
    "created_at": "2025-01-06T10:00:00Z",
    "updated_at": "2025-01-06T10:00:00Z"
  }
}
```

### 2. Update User Preferences
**PUT** `/api/v1/user/preferences`

Updates user preferences. Supports partial updates - only include fields you want to change.

**Request Body:**
```json
{
  "beginner_mode": true,
  "tooltip_delay": 1000,
  "preferred_sports": ["nfl", "mlb", "nba"],
  "ai_suggestions_enabled": false
}
```

**Validation Rules:**
- `tooltip_delay`: Must be between 0 and 5000 milliseconds
- `preferred_sports`: Must be valid sports: "nfl", "nba", "mlb", "nhl", "pga", "nascar", "mma", "soccer"

**Example Response:**
```json
{
  "success": true,
  "data": {
    "user_id": 123,
    "beginner_mode": true,
    "show_tooltips": true,
    "tooltip_delay": 1000,
    "preferred_sports": ["nfl", "mlb", "nba"],
    "ai_suggestions_enabled": false,
    "created_at": "2025-01-06T10:00:00Z",
    "updated_at": "2025-01-06T10:15:00Z"
  }
}
```

### 3. Reset Preferences to Defaults
**POST** `/api/v1/user/preferences/reset`

Resets all user preferences to their default values.

**Default Values:**
- `beginner_mode`: false
- `show_tooltips`: true
- `tooltip_delay`: 500
- `preferred_sports`: []
- `ai_suggestions_enabled`: true

**Example Response:**
```json
{
  "success": true,
  "data": {
    "user_id": 123,
    "beginner_mode": false,
    "show_tooltips": true,
    "tooltip_delay": 500,
    "preferred_sports": [],
    "ai_suggestions_enabled": true,
    "created_at": "2025-01-06T10:00:00Z",
    "updated_at": "2025-01-06T10:20:00Z"
  }
}
```

## Error Responses

### Unauthorized (401)
```json
{
  "success": false,
  "error": {
    "code": "UNAUTHORIZED",
    "message": "User ID not found in context"
  }
}
```

### Validation Error (400)
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "Invalid tooltip delay",
    "details": "Tooltip delay must be between 0 and 5000 milliseconds"
  }
}
```

### Internal Server Error (500)
```json
{
  "success": false,
  "error": {
    "code": "INTERNAL_ERROR",
    "message": "Failed to update preferences"
  }
}
```

## Testing with cURL

### Get preferences:
```bash
curl -X GET http://localhost:8080/api/v1/user/preferences \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Update preferences:
```bash
curl -X PUT http://localhost:8080/api/v1/user/preferences \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "beginner_mode": true,
    "tooltip_delay": 1000,
    "preferred_sports": ["nfl", "nba"]
  }'
```

### Reset preferences:
```bash
curl -X POST http://localhost:8080/api/v1/user/preferences/reset \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```