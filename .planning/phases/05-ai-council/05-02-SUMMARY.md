# Phase 5 Plan 2: External API Validator Summary

**Implemented ExternalValidator with Claude and OpenAI API integration, retry logic, and fail-closed error handling.**

## Accomplishments

- ExternalValidator interfaces with Claude 3.5 Sonnet and OpenAI GPT-4 Turbo
- validate_parallel() runs both APIs concurrently via asyncio.gather
- Retry logic with exponential backoff (max 2 retries)
- Fail-closed on all errors (network, auth, rate limit, timeout)
- Missing API keys handled gracefully

## Files Created/Modified

- `core/council/external_validator.py` - External API validation interface
- `core/council/__init__.py` - Module exports (added ExternalValidator)
- `core/council/requirements.txt` - Added anthropic==0.18.1, openai==1.12.0

## Decisions Made

- Claude 3.5 Sonnet chosen (superior reasoning per research)
- Parallel API calls via asyncio.gather (2-5 seconds total)
- Max 2 retries with exponential backoff (connection errors only)
- Fail-closed on rate limits (don't hammer APIs)
- Authentication errors fail immediately (misconfiguration)
- Synchronous SDK calls wrapped in run_in_executor for async compatibility

## Implementation Details

### ExternalValidator
- `validate_with_claude(decision)` - Async Claude 3.5 Sonnet validation
- `validate_with_openai(decision)` - Async GPT-4 Turbo validation
- `validate_parallel(decision)` - Concurrent validation via asyncio.gather
- `_retry_with_backoff()` - Retry helper with exponential backoff
- `_build_validation_prompt()` - Shared prompt construction
- `_parse_response()` - Extract confidence and critique from responses

### Error Handling
- Connection errors: Retry up to 2 times with exponential backoff
- Authentication errors: Fail immediately (not transient)
- Rate limit errors: Fail immediately (don't hammer APIs)
- Timeout errors: Retry with backoff, then fail-closed
- Missing API keys: Log warning, skip that API, continue

### Environment Variables
- `ANTHROPIC_API_KEY` - API key for Claude (optional)
- `OPENAI_API_KEY` - API key for OpenAI (optional)

## Issues Encountered

None

## Next Step

Ready for 05-03-PLAN.md (Consensus Aggregator with confidence-weighted voting)
