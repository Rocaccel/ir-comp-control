#include <Arduino.h>
#include <IRremote.hpp>

// Пин, к которому подключен выход (OUT) ИК-приемника
const int RECV_PIN = 2; 

void setup() {
    Serial.begin(115200); // Скорость должна совпадать с baud в твоем config.yaml
    Serial.setTimeout(5);
    // Запуск приемника
    IrReceiver.begin(RECV_PIN); 
    
    Serial.println("Arduino IR to Serial Bridge Ready (RC5)");
}

void loop() {
    if (IrReceiver.decode()) {
        // Проверяем, что сигнал распознан
        if (IrReceiver.decodedIRData.protocol != UNKNOWN) {
            
            /* В протоколе RC5 есть "toggle bit" (бит переключения), 
               который меняется при каждом новом нажатии. 
               Нам нужен только номер команды (command).
            */
            uint16_t command = IrReceiver.decodedIRData.command;

            // Вывод в формате, который понимает наш Go-скрипт
            Serial.print("KEY_");
            Serial.println(command, HEX); // Вывод кода в шестнадцатеричном формате
        }
        
        // Готовимся принимать следующий сигнал
        IrReceiver.resume(); 
    }
}
