#include <Arduino.h>
#include <IRremote.hpp>

const int RECV_PIN = 2;

void setup() {
    Serial.begin(115200);
    Serial.setTimeout(5);
    IrReceiver.begin(RECV_PIN);
}

void loop() {
    if (IrReceiver.decode()) {
        const char* prefix = nullptr;
        switch (IrReceiver.decodedIRData.protocol) {
            case NEC:
            case NEC2:
                prefix = "NEC_";
                break;
            case RC5:
                prefix = "RC5_";
                break;
            default:
                break; // прочие протоколы не поддерживаем — не печатаем
        }
        if (prefix != nullptr) {
            Serial.print(prefix);
            Serial.println(IrReceiver.decodedIRData.command, HEX);
        }
        IrReceiver.resume();
    }
}
