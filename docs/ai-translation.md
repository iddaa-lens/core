# AI Translation for League Matching

## Overview

The AI Translation service provides intelligent translation of Turkish league names to English for better matching with the Football API. This significantly improves the accuracy of league mapping compared to static translation tables.

## Features

- **Dynamic Translation**: Handles any Turkish league name using OpenAI GPT-3.5-turbo
- **Context Awareness**: Considers country and league type for better translations
- **Multiple Variations**: Generates 3-5 English variations per Turkish name
- **Smart Caching**: In-memory cache prevents duplicate API calls
- **Graceful Fallback**: Falls back to static translation if AI service fails
- **Cost Efficient**: Uses low-cost GPT-3.5-turbo with minimal tokens

## Configuration

Set the OpenAI API key as an environment variable:

```bash
export OPENAI_API_KEY="your-openai-api-key-here"
```

If not set, the system automatically falls back to static translation.

## Usage

The AI translation is automatically used in the leagues sync job:

```bash
# Run leagues sync with AI translation
export DATABASE_URL="postgresql://iddaa:iddaa123@localhost:5433/iddaa_core?sslmode=disable"
export FOOTBALL_API_KEY="your-football-api-key"
export OPENAI_API_KEY="your-openai-api-key"

go run cmd/cron/main.go -job leagues -once
```

## Translation Examples

| Turkish League Name   | AI Translations                                       |
| --------------------- | ----------------------------------------------------- |
| Türkiye Süper Lig     | Super Lig, Turkish Super League, Turkey Super League  |
| UEFA Şampiyonlar Ligi | Champions League, UEFA Champions League, European Cup |
| İspanya La Liga       | La Liga, Spanish La Liga, Primera Division            |
| Almanya Bundesliga    | Bundesliga, German Bundesliga                         |

## Implementation Details

### AI Translation Service

```go
type AITranslationService struct {
    client   *http.Client
    apiKey   string
    baseURL  string
    cache    map[string][]string
}
```

### Translation Process

1. **Cache Check**: First checks if translation already cached
2. **AI Call**: Sends focused prompt to OpenAI API
3. **Response Parsing**: Extracts multiple English variations
4. **Cache Storage**: Stores result for future use
5. **Fallback**: Uses static translation if AI fails

### Prompt Engineering

The service uses a carefully crafted prompt that:

- Focuses on football/soccer terminology
- Requests multiple variations for better matching
- Provides examples for consistency
- Asks for API-friendly names

## Cost Analysis

- **Model**: GPT-3.5-turbo ($0.0015 per 1K input tokens, $0.002 per 1K output tokens)
- **Typical Cost**: ~$0.001 per league translation
- **Cache Benefits**: Each league translated only once
- **Total Cost**: <$1 for translating all Turkish leagues

## Error Handling

- **API Failures**: Graceful fallback to static translation
- **Rate Limits**: Built-in timeout and retry logic
- **Invalid Responses**: Robust parsing with fallbacks
- **Network Issues**: Timeout protection with context cancellation

## Monitoring

Check logs for translation activity:

```
2025/06/03 22:48:56 AI translated 'Türkiye Süper Lig' to: [Super Lig, Turkish Super League, Turkey Super League]
2025/06/03 22:48:56 Using cached translation for: UEFA Şampiyonlar Ligi
```

## Benefits over Static Translation

1. **Accuracy**: AI understands context and nuance
2. **Coverage**: Handles any league name, including new ones
3. **Maintenance**: No need to manually update translation tables
4. **Adaptability**: Can be easily modified for other languages
5. **Quality**: Generates multiple high-quality variations

## Testing

Test the AI translation service:

```bash
export OPENAI_API_KEY="your-key"
go run /tmp/test_ai_translation.go
```

This will translate several example league names and show the results.
