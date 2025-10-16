import tensorflow as tf
from tensorflow.keras import layers, models

num_classes = 4  # e.g. silence, traffic, rain, talking

model = models.Sequential([
    layers.Input(shape=(64, 64, 1)),  # (mel bands, time, channel)
    layers.Conv2D(32, (3,3), activation='relu'),
    layers.MaxPooling2D((2,2)),
    layers.Conv2D(64, (3,3), activation='relu'),
    layers.MaxPooling2D((2,2)),
    layers.Flatten(),
    layers.Dense(64, activation='relu'),
    layers.Dense(num_classes, activation='softmax')
])

model.compile(optimizer='adam',
              loss='categorical_crossentropy',
              metrics=['accuracy'])
