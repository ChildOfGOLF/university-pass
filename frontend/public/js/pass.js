//Логика для охраны
import QRCode from "https://esm.sh/qrcode@1.5.3"
import { createTOTP, msUntilNext } from "./totp.js";

const secretKey = localStorage.getItem('secret_key');

const errorElement = document.getElementById("pass-error");
const contentElement = document.getElementById("pass-content");
const canvas = document.getElementById("qr-canvas");
const timerElement = document.getElementById("timer-text");
const logoutButton = document.getElementById("logout-button");

if (!secretKey) {
    window.location.href = "index.html";
}

function showError(msg) {
    errorElement.textContent = msg;
    contentElement.hidden = true
}

let totp;
try {
    totp = createTOTP(secretKey);
}
catch (err) {
    showError("Что-то не получилось");
}

async function renderPass() {
    try {
        let code = totp.generate();
        let codeToCanvas = code.toString() + localStorage.getItem("device_id").toString();
        await QRCode.toCanvas(canvas, codeToCanvas, {width: 201});
        errorElement.textContent = "";
        contentElement.hidden = false;
        let device_id = localStorage.getItem("device_id");
        console.log(`${device_id}, ${code}, ${secretKey}, ${codeToCanvas}`);
    }
    catch (err) {
        showError("Не удалось построить QR-код");
    }
}

//Тут обновление куара
function refreshWindow() {
    setTimeout(() => {
        renderPass(),
        refreshWindow();
    },    msUntilNext());
}

function tickiTickiTick () {
    const secondsLeft = Math.ceil(msUntilNext() / 1000);
    timerElement.textContent = secondsLeft == 0 ? 30 : secondsLeft;
}

logoutButton.addEventListener("click", () => {
    localStorage.removeItem("secret_key");
    window.location.href = "index.html";
})

if (totp) {
    renderPass();
    refreshWindow();
    tickiTickiTick();
    setInterval(tickiTickiTick, 1000);
}

