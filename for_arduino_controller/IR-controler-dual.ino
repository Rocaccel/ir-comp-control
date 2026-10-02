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
        if (IrReceiver.decodedIRData.protocol != UNKNOWN) {
            uint16_t command = IrReceiver.decodedIRData.command;
            const char* prefix = "KEY_";

            switch (IrReceiver.decodedIRData.protocol) {
                case NEC:
                case NEC2:
                    prefix = "NEC_";
                    break;
                case RC5:
                    prefix = "RC5_";
                    break;
                default:
                    break;
            }

            Serial.print(prefix);
            Serial.println(command, HEX);
        }
        IrReceiver.resume();
    }
}
