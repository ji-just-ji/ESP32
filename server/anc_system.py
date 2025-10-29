import numpy as np
from config import FILTER_LENGTH, MU

class AdaptiveANC:
    def __init__(self):
        self.filter_length = FILTER_LENGTH
        self.mu = MU
        self.w = np.zeros(self.filter_length)
        self.x_buffer = np.zeros(self.filter_length)
    
    def process_chunk(self, x_ref: np.ndarray, d_error: np.ndarray) -> np.ndarray:
        """
        NLMS algorithm for adaptive filtering (no secondary path).
        x_ref: Reference microphone signal
        d_error: Desired/error signal  
        Returns: Filter output (anti-noise)
        """
        y_out = np.zeros(len(x_ref))
        
        for n in range(len(x_ref)):
            # Update reference buffer
            self.x_buffer[:-1] = self.x_buffer[1:]
            self.x_buffer[-1] = x_ref[n]
            
            # Filter output: y(n) = w^T * x(n)
            y_out[n] = np.dot(self.w, self.x_buffer)
            
            # Error signal
            e = d_error[n]
            
            # Compute NLMS step
            power = np.dot(self.x_buffer, self.x_buffer) + 1e-8
            self.w = self.w + (self.mu / power) * e * self.x_buffer
        
        return y_out
