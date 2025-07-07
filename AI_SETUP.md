# AI Features Setup Guide

## Overview

The DFS Lineup Optimizer includes AI-powered features for player recommendations and lineup analysis. These features use Claude (Anthropic's AI) to provide intelligent suggestions based on your lineup constraints and contest type.

## Configuration

### 1. Get an Anthropic API Key

1. Visit [Anthropic Console](https://console.anthropic.com/)
2. Create an account or sign in
3. Navigate to API Keys section
4. Create a new API key
5. Copy the key (it starts with `sk-ant-api03-...`)

### 2. Configure Your Environment

Add the API key to your `.env` file:

```bash
# AI Configuration
ANTHROPIC_API_KEY=sk-ant-api03-your-key-here
```

### 3. Restart the Backend

After adding the API key, restart the backend service:

```bash
docker-compose restart backend
```

## Available AI Features

### Player Recommendations

Get AI-powered player recommendations based on:
- Contest type (GPP vs Cash)
- Remaining budget
- Positions needed
- Current lineup composition
- Optimization strategy (ceiling vs floor)

**Endpoint**: `POST /api/v1/ai/recommend-players`

### Lineup Analysis

Analyze your lineup to get:
- Overall score and assessment
- Strengths and weaknesses
- Improvement suggestions
- Stacking analysis
- Risk level assessment

**Endpoint**: `POST /api/v1/ai/analyze-lineup`

### Recommendation History

View past AI recommendations:

**Endpoint**: `GET /api/v1/ai/recommendations/history`

## Frontend Usage

The AI features are integrated into the lineup builder interface:

1. Click the "AI Assistant" button in the optimizer panel
2. Click "Generate Recommendations" to get player suggestions
3. The AI will analyze your current lineup and suggest players based on:
   - Your remaining budget
   - Positions you need to fill
   - Contest type optimization strategy

## Troubleshooting

### CORS Errors
- The frontend now properly routes all AI requests through the backend proxy
- No direct browser-to-Anthropic API calls are made

### 404 Errors
- Ensure you've restarted the backend after configuration changes
- Check that the AI routes are registered in the logs

### API Key Issues
- Verify your API key is valid and has sufficient credits
- Check the backend logs for specific error messages
- Ensure the key is properly formatted in the `.env` file

## Security Notes

- The Anthropic API key is stored securely on the backend only
- All AI requests are proxied through the backend API
- The frontend never has direct access to the API key
- Rate limiting is implemented to prevent abuse

## Development Notes

For development, the AI endpoints are temporarily configured to work without authentication. In production:
1. Re-enable authentication in `/backend/internal/api/router.go`
2. Update the AI handlers to require proper authentication
3. Ensure proper user context is passed to AI requests