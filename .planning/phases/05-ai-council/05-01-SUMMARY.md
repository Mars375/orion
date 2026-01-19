# Phase 5 Plan 1: Local SLM Validator Summary

**Implemented CouncilValidator with Ollama integration and Pi 5-aware resource monitoring.**

## Accomplishments

- CouncilValidator interfaces with Ollama-served Gemma-2 2B model
- MemoryManager enforces 4GB minimum free RAM before loading
- Temperature monitoring implemented (optional, non-blocking)
- 30-second timeout per validation
- Fail-closed error handling

## Files Created/Modified

- `core/council/council_validator.py` - Local SLM validation interface
- `core/council/memory_manager.py` - Resource monitoring for Pi 5
- `core/council/__init__.py` - Module exports
- `core/council/requirements.txt` - Added ollama==0.1.6, psutil==5.9.8

## Decisions Made

- Gemma-2 2B chosen over Phi-3 (lower RAM: 3GB vs 5GB)
- Ollama Python client over subprocess calls (cleaner error handling)
- Temperature monitoring non-blocking (availability varies by environment)
- 4GB free RAM threshold (conservative, leaves margin for other services)
- `set_memory_manager()` pattern for optional resource monitoring integration

## Implementation Details

### CouncilValidator
- `validate(decision)` returns `Tuple[float, str]` (confidence, critique)
- Builds validation prompt asking model to evaluate SAFE/RISKY classification
- Parses structured response (CONFIDENCE: X.X, CRITIQUE: ...)
- Integrates with MemoryManager via `set_memory_manager()` method
- Uses `ollama.generate()` with 30s timeout

### MemoryManager
- `check_resources_before_load()` returns `Tuple[bool, str]`
- `get_free_ram_gb()` uses psutil for cross-platform RAM monitoring
- `get_cpu_temperature()` tries vcgencmd (Pi) then psutil sensors
- `monitor_during_inference()` context manager tracks resource usage

## Issues Encountered

None

## Next Step

Ready for 05-02-PLAN.md (External Validator with Claude + OpenAI APIs)
