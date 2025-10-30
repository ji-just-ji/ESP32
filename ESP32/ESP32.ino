#include <driver/i2s.h>
#include <WiFi.h>
#include "ESP32MQTTClient.h"
#include "esp_idf_version.h"

#define SAMPLE_BUFFER_SIZE 4096
#define SAMPLE_RATE 16000

#define I2S_MIC_SERIAL_CLOCK 2
#define I2S_MIC_LEFT_RIGHT_CLOCK 15
#define I2S_MIC 13
#define I2S_SPEAKER_SERIAL_CLOCK 26
#define I2S_SPEAKER_LEFT_RIGHT_CLOCK 27
#define I2S_SPEAKER 25

const char *ssid = "username";
const char *password = "password";

char *server = "mqtt://<ip>";

char *subscribeTopic = "audio/room/speaker";
char *audioTopic = "sensor/room/audio";

ESP32MQTTClient mqttClient;

// Microphone I2S configuration
i2s_config_t i2s_mic_config = {
    .mode = (i2s_mode_t)(I2S_MODE_MASTER | I2S_MODE_RX),
    .sample_rate = SAMPLE_RATE, 
    .bits_per_sample = I2S_BITS_PER_SAMPLE_32BIT, // Microphone takes in 24bit audio
    .channel_format = I2S_CHANNEL_FMT_ONLY_LEFT,
    .communication_format = I2S_COMM_FORMAT_I2S,
    .intr_alloc_flags = ESP_INTR_FLAG_LEVEL1,
    .dma_buf_count = 4,
    .dma_buf_len = 1024,
    .use_apll = false,
    .tx_desc_auto_clear = false,
    .fixed_mclk = 0
};

i2s_pin_config_t i2s_mic_pins = {
    .bck_io_num = I2S_MIC_SERIAL_CLOCK,
    .ws_io_num = I2S_MIC_LEFT_RIGHT_CLOCK,
    .data_out_num = I2S_PIN_NO_CHANGE,
    .data_in_num = I2S_MIC
};

// Speaker I2S Configuration
i2s_config_t i2s_speaker_config = {
    .mode = (i2s_mode_t)(I2S_MODE_MASTER | I2S_MODE_TX),
    .sample_rate = SAMPLE_RATE, 
    .bits_per_sample = I2S_BITS_PER_SAMPLE_16BIT,
    .channel_format = I2S_CHANNEL_FMT_RIGHT_LEFT,
    .communication_format = I2S_COMM_FORMAT_I2S,
    .intr_alloc_flags = ESP_INTR_FLAG_LEVEL1,
    .dma_buf_count = 8,
    .dma_buf_len = 1024,
    .use_apll = false,
    .tx_desc_auto_clear = true,
    .fixed_mclk = 0
};

i2s_pin_config_t i2s_speaker_pins = {
    .bck_io_num = I2S_SPEAKER_SERIAL_CLOCK,
    .ws_io_num = I2S_SPEAKER_LEFT_RIGHT_CLOCK,
    .data_out_num = I2S_SPEAKER,
    .data_in_num = I2S_PIN_NO_CHANGE
};

void setup() {
    Serial.begin(115200);

    // log_i();
    // log_i("setup, ESP.getSdkVersion(): ");
    // log_i("%s", ESP.getSdkVersion());

    // mqttClient.enableDebuggingMessages();

    // mqttClient.setURI(server);
    // mqttClient.enableLastWillMessage("lwt", "I am going offline");
    // mqttClient.setKeepAlive(30);
    // mqttClient.setOnMessageCallback([](const std::string &topic, const std::string &payload) {
    //     log_i("Global callback: %s: %s", topic.c_str(), payload.c_str());
    // });
    // WiFi.begin(ssid, password);
    // Serial.print("Connecting to WiFi");
    // while (WiFi.status() != WL_CONNECTED) {
    //     delay(500);
    //     Serial.print(".");
    // }
    // Serial.println("\nWiFi connected. IP: " + WiFi.localIP().toString());
    // mqttClient.loopStart();

    i2s_driver_install(I2S_NUM_0, &i2s_mic_config, 0, NULL);
    i2s_set_pin(I2S_NUM_0, &i2s_mic_pins);

    i2s_driver_install(I2S_NUM_1, &i2s_speaker_config, 0, NULL);
    i2s_set_pin(I2S_NUM_1, &i2s_speaker_pins);

}

int32_t mic_samples[SAMPLE_BUFFER_SIZE];
int16_t speaker_samples[SAMPLE_BUFFER_SIZE];

void loop() {

    // Mic reading
    size_t bytes_read = 0;
    i2s_read(I2S_NUM_0, mic_samples, sizeof(int32_t) * SAMPLE_BUFFER_SIZE, &bytes_read, portMAX_DELAY);

    // int samples_read = bytes_read / sizeof(int32_t);
    // for (int i = 0; i < samples_read; i++)
    // {
    //     Serial.printf("%ld\n", mic_samples[i]);
    // }

    // if (bytes_read > 0) {
    //     mqrrClient.publish(audioTopic, mic_samples, 0, false); // need adjusting
    // }

    // Down from 32 bit to 16 bit
    for (int i = 0; i < SAMPLE_BUFFER_SIZE; i++) {
        int32_t sample24 = mic_samples[i] >> 8;
        speaker_samples[i] = sample24 >> 8;
    }

    size_t bytes_written = 0;
    i2s_write(I2S_NUM_1, speaker_samples, sizeof(int16_t) * SAMPLE_BUFFER_SIZE, &bytes_written, portMAX_DELAY);

}


void onMqttConnect(esp_mqtt_client_handle_t client) {
    if (mqttClient.isMyTurn(client)) {
        mqttClient.subscribe(subscribeTopic, [](const std::string &payload)
                                { log_i("%s: %s", subscribeTopic, payload.c_str()); });
    }
}

#if ESP_IDF_VERSION < ESP_IDF_VERSION_VAL(5, 0, 0)
esp_err_t handleMQTT(esp_mqtt_event_handle_t event) {
    mqttClient.onEventCallback(event);
    return ESP_OK;
}
#else
void handleMQTT(void *handler_args, esp_event_base_t base, int32_t event_id, void *event_data) {
    auto *event = static_cast<esp_mqtt_event_handle_t>(event_data);
    mqttClient.onEventCallback(event);
}
#endif