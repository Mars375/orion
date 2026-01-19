"""
ORION Memory Manager - Resource monitoring for Pi 5 constraints.

Monitors system resources before and during local model inference to prevent
OOM crashes and thermal throttling on resource-constrained hardware.

Raspberry Pi 5 Constraints:
- 8GB RAM total, need ~4GB free for safe model loading
- Gemma-2 2B requires ~3GB during inference
- CPU temperature must stay below 70°C for sustained performance
- No swap usage for inference (too slow)

Safety Invariants:
- Block model loading if < 4GB RAM free
- Warn (don't block) if temperature > 70°C
- Log resource usage during inference
- Temperature monitoring is optional (vcgencmd may not be available)
"""

import logging
import subprocess
import time
from contextlib import contextmanager
from typing import Tuple, Generator

import psutil


logger = logging.getLogger(__name__)


# Minimum free RAM required before loading model (GB)
MIN_FREE_RAM_GB = 4.0

# Temperature warning threshold (Celsius)
TEMPERATURE_WARNING_THRESHOLD = 70.0


class MemoryManager:
    """
    Manages system resources for local SLM inference on Pi 5.

    Provides resource checks before model loading and monitoring during
    inference to prevent OOM crashes and thermal issues.

    Invariants:
    - RAM check is mandatory (blocks if insufficient)
    - Temperature check is advisory (warns but doesn't block)
    - Temperature monitoring is optional (gracefully handles missing vcgencmd)
    - All checks are non-destructive (read-only system queries)
    """

    def __init__(
        self,
        min_free_ram_gb: float = MIN_FREE_RAM_GB,
        temperature_threshold: float = TEMPERATURE_WARNING_THRESHOLD,
    ) -> None:
        """
        Initialize memory manager.

        Args:
            min_free_ram_gb: Minimum free RAM in GB required before loading model
            temperature_threshold: CPU temperature threshold for warnings (Celsius)
        """
        self.min_free_ram_gb = min_free_ram_gb
        self.temperature_threshold = temperature_threshold

        logger.info(
            f"MemoryManager initialized: min_ram={min_free_ram_gb}GB, "
            f"temp_threshold={temperature_threshold}°C"
        )

    def get_free_ram_gb(self) -> float:
        """
        Get available RAM in gigabytes.

        Returns:
            Available RAM in GB
        """
        mem = psutil.virtual_memory()
        available_gb = mem.available / (1024 ** 3)
        return available_gb

    def get_cpu_temperature(self) -> Tuple[float, bool]:
        """
        Get CPU temperature if available.

        Uses vcgencmd on Raspberry Pi, falls back to psutil sensors.

        Returns:
            Tuple of (temperature_celsius, is_available)
            If temperature cannot be read, returns (0.0, False)
        """
        # Try vcgencmd first (Raspberry Pi specific)
        try:
            result = subprocess.run(
                ["vcgencmd", "measure_temp"],
                capture_output=True,
                text=True,
                timeout=5,
            )
            if result.returncode == 0:
                # Output format: temp=45.0'C
                temp_str = result.stdout.strip()
                temp_value = float(temp_str.split("=")[1].replace("'C", ""))
                return (temp_value, True)
        except (subprocess.TimeoutExpired, FileNotFoundError, ValueError, IndexError):
            pass

        # Fallback to psutil sensors
        try:
            temps = psutil.sensors_temperatures()
            if temps:
                # Try common sensor names
                for name in ["cpu_thermal", "coretemp", "cpu-thermal"]:
                    if name in temps and temps[name]:
                        return (temps[name][0].current, True)
                # Use first available sensor
                for sensor_list in temps.values():
                    if sensor_list:
                        return (sensor_list[0].current, True)
        except Exception:
            pass

        logger.debug("CPU temperature monitoring not available")
        return (0.0, False)

    def check_resources_before_load(self) -> Tuple[bool, str]:
        """
        Check if system has sufficient resources to load a model.

        Checks:
        1. Available RAM (mandatory, blocks if insufficient)
        2. CPU temperature (advisory, logs warning if high)

        Returns:
            Tuple of (can_proceed, message):
            - can_proceed: True if model can be safely loaded
            - message: Description of resource status or reason for blocking

        Safety:
            - Blocks if RAM < min_free_ram_gb
            - Warns but doesn't block for high temperature
        """
        # Check RAM (mandatory)
        free_ram = self.get_free_ram_gb()

        if free_ram < self.min_free_ram_gb:
            reason = (
                f"Insufficient RAM: {free_ram:.1f}GB free, "
                f"need {self.min_free_ram_gb}GB minimum"
            )
            logger.warning(reason)
            return (False, reason)

        # Check temperature (advisory)
        temp, temp_available = self.get_cpu_temperature()

        if temp_available and temp > self.temperature_threshold:
            logger.warning(
                f"CPU temperature elevated: {temp:.1f}°C "
                f"(threshold: {self.temperature_threshold}°C)"
            )

        # Log resource status
        if temp_available:
            status = f"Resources OK: {free_ram:.1f}GB RAM free, CPU temp {temp:.1f}°C"
        else:
            status = f"Resources OK: {free_ram:.1f}GB RAM free (temp monitoring unavailable)"

        logger.info(status)
        return (True, status)

    @contextmanager
    def monitor_during_inference(self) -> Generator[None, None, None]:
        """
        Context manager to monitor resources during model inference.

        Tracks resource usage and logs warnings if temperature rises above
        threshold during inference.

        Usage:
            with memory_manager.monitor_during_inference():
                # Perform inference
                response = ollama.generate(...)

        Yields:
            None

        Safety:
            - Non-blocking: Only logs warnings, doesn't interrupt inference
            - Time tracking: Measures inference duration
        """
        start_time = time.time()
        start_temp, temp_available = self.get_cpu_temperature()
        start_ram = self.get_free_ram_gb()

        logger.debug(
            f"Inference started: RAM={start_ram:.1f}GB free"
            + (f", temp={start_temp:.1f}°C" if temp_available else "")
        )

        try:
            yield
        finally:
            end_time = time.time()
            duration = end_time - start_time
            end_temp, _ = self.get_cpu_temperature()
            end_ram = self.get_free_ram_gb()

            # Calculate changes
            ram_used = start_ram - end_ram

            log_msg = f"Inference completed in {duration:.1f}s, RAM delta: {ram_used:+.1f}GB"

            if temp_available:
                temp_delta = end_temp - start_temp
                log_msg += f", temp delta: {temp_delta:+.1f}°C"

                if end_temp > self.temperature_threshold:
                    logger.warning(
                        f"Temperature exceeded threshold during inference: "
                        f"{end_temp:.1f}°C > {self.temperature_threshold}°C"
                    )

            logger.info(log_msg)
