//Логика для охраны

import { apiMethods } from "./api.js";

const qrScannerButton = document.getElementById('scanner-button');

qrScannerButton.addEventListener('click', async () => {
    try {
        /* const res = await apiMethods.verify({
            device_id: "",
            direction: "enter",
            guest_id: "550e8400-e29b-41d4-a716-446655440000",
            otp: ""
        });

        console.log('click');
        console.log(res); */

        const { secret_key } = await apiMethods.login({
            device_id: "device-123456",
            email: "student1@uni.com",
            password: "password123"
        });

        console.log(secret_key);
    }
    catch (err) {
        console.log(`${err.name}: ${err.massege}`);
    }
});