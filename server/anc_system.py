import numpy as np
from config import FILTER_LENGTH, MU, LATENCY_SAMPLES, CHUNK_SIZE

class AdaptiveANC:
    def __init__(self):
        self.filter_length = FILTER_LENGTH
        self.mu = MU
        self.w = np.zeros(self.filter_length)
        self.x_buffer = np.zeros(self.filter_length + LATENCY_SAMPLES)
        
        # Performance monitoring
        self.error_history = []
        self.convergence_threshold = 0.01
        self.window_size = 100
        
        # IoT specific additions
        self.last_chunk_time = None
        self.packet_count = 0
        self.signal_threshold = 0.1  # Minimum signal level to process
        self.max_adaptation_rate = 0.1  # Limit maximum weight change
        
    def is_converged(self):
        if len(self.error_history) < self.window_size:
            return False
        recent_errors = self.error_history[-self.window_size:]
        error_variance = np.var(recent_errors)
        return error_variance < self.convergence_threshold
    
    def process_chunk(self, x_ref: np.ndarray, d_error: np.ndarray) -> np.ndarray:
        # Basic signal validation for ESP32 ADC readings
        if np.abs(x_ref).mean() < self.signal_threshold or np.abs(d_error).mean() < self.signal_threshold:
            return np.zeros_like(d_error)  # Return silence if signal too weak
            
        # Remove any DC offset from ESP32 ADC
        x_ref = x_ref - np.mean(x_ref)
        d_error = d_error - np.mean(d_error)
        
        # Shift and update reference buffer
        self.x_buffer = np.roll(self.x_buffer, -len(x_ref))
        self.x_buffer[-len(x_ref):] = x_ref
        
        # Use delayed reference for processing
        x_delayed = self.x_buffer[:-LATENCY_SAMPLES][-self.filter_length:]
        
        # Calculate signal power for variable step size
        signal_power = np.mean(x_delayed ** 2)
        mu_adjusted = min(self.mu / (signal_power + 1e-6), self.max_adaptation_rate)
        
        # Adaptive filtering with delayed reference
        y = np.dot(self.w, x_delayed)
        e = d_error - y
        
        # Update weights with leakage term (prevents unbounded growth)
        leakage = 0.9999
        weight_update = 2 * mu_adjusted * e * x_delayed
        
        # Limit maximum weight change per iteration (for stability with network jitter)
        max_update = 0.1
        weight_update = np.clip(weight_update, -max_update, max_update)
        
        self.w = leakage * self.w + weight_update
        
        # Constrain maximum weight values
        max_weight = 2.0
        np.clip(self.w, -max_weight, max_weight, out=self.w)
        
        # Normalize output for ESP32 DAC
        if np.max(np.abs(e)) > 1.0:
            e = e / np.max(np.abs(e))
        
        # Smooth transitions to prevent speaker damage
        e = np.tanh(e)  # Soft limiting
        
        self.error_history.append(np.mean(e**2))
        if len(self.error_history) > self.window_size * 2:
            self.error_history = self.error_history[-self.window_size:]
        
        self.packet_count += 1
        return e