import librosa
import numpy as np
import matplotlib.pyplot as plt

y, sr = librosa.load("output.wav", sr=16000)
mel_spec = librosa.feature.melspectrogram(y=y, sr=sr, n_mels=64)
mel_db = librosa.power_to_db(mel_spec, ref=np.max)

plt.figure(figsize=(10, 4))
librosa.display.specshow(mel_db, sr=sr, x_axis='time', y_axis='mel')
plt.title('Mel Spectrogram')
plt.colorbar(format='%+2.0f dB')
plt.show()
