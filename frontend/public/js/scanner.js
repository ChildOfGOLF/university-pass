//Логика для scanner

import QrScanner from "./QrScanner/qr-scanner.min.js";
import { apiMethods } from "./api.js";

const video = document.getElementById('camera');
const fileInput = document.getElementById('file-input');
const restartButton = document.getElementById('restart-button');
const errorText = document.getElementById('error-text');
const acceptDenyHolder = document.getElementById('accept-deny-holder');
const acceptDenyMsg = document.getElementById('accept-deny-message');
const acceptDenyImg = document.getElementById('accept-deny-img');
const acceptDenyText = document.getElementById('accept-deny-text');
const reason = document.getElementById('reason');
const tableInfo = document.getElementById('table-info');
const cancelButton = document.getElementById('cancel-button');

function stopScanning(scanner) {
    scanner.stop();
    restartButton.style.display = 'flex';
}

function setError(text) {
    errorText.textContent = text;
}

function parseQrData(data) {

    const result = {
        device_id: "",
        guest_id: "",
        otp: ""
    };

    const uuidRegexp =
        /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

    if (uuidRegexp.test(data)) {
        result.guest_id = data;

        return result;
    }

    const otpRegexp = /^(\d{6})(.+)$/i;
    const match = data.match(otpRegexp);

    if (match) {
        result.otp = match[1];
        result.device_id = match[2];

        return result;
    }

    const newErr = new Error("Не валидный QR");
    newErr.name = "InvalidQr";

    throw newErr;
}

function hideAcceptDenyWindow(status) {
    let timer = 500;
    let className = 'accept'

    if (!status) {
        timer = 1600;
        className = 'deny';
    }

    setTimeout(() => {
        acceptDenyHolder.style.display = 'none';
        acceptDenyImg.src = "";
        acceptDenyText.textContent = "";
        reason.textContent = "";

        acceptDenyMsg.classList.remove(className);
    }, timer);
}

function showAcceptDenyWindow(status) {
    let src = "";
    let text = "";

    if (status) {
        src = "../icons/scanner/accept.png";
        text = "Разрешено";

        acceptDenyMsg.classList.add('accept');
    }
    else {
        src = "../icons/scanner/deny.png";
        text = "Отклонено";

        acceptDenyMsg.classList.add('deny');
    }

    acceptDenyImg.src = src;
    acceptDenyText.textContent = text;
    acceptDenyHolder.style.display = 'flex';

    hideAcceptDenyWindow(status);
}

function fillTable(visitorInfo) {
    showAcceptDenyWindow(true);

    const person = visitorInfo.user ?? visitorInfo.guest;
    tableInfo.querySelectorAll('tr[data-attribute]')
        .forEach(tr => {
            const info = person?.[tr.dataset.attribute] ?? "-";
            tr.querySelector('td:last-child').textContent = info;
        });
}

function clearTable() {
    tableInfo.querySelectorAll('tr')
        .forEach(tr => {
            tr.querySelector('td:last-child').textContent = "-";
        });
}

async function setResult(result) {
    clearTable();
    try {
        const data = parseQrData(result);
        // console.log(data);

        const res = await apiMethods.verify(data);
        console.log(res);

        if (res.is_allowed) {
            showAcceptDenyWindow(true);
            fillTable(res);
        }
        else {
            reason.textContent = res.reason;
            showAcceptDenyWindow(false);
        }
    }
    catch (err) {
        if (err.name === "InvalidQr") {
            reason.textContent = err.message;
            showAcceptDenyWindow(false);

            return;
        }
        console.log(err);
        setError("Ошибка");
    }
}

const scanner = new QrScanner(video, result => { stopScanning(scanner); setResult(result.data); }, {
    onDecodeError: error => {
        if (error.name !== error.NO_QR_CODE_FOUND)
            console.log(`error: ${error}`);
    },
    highlightScanRegion: true,
    highlightCodeOutline: true
});

async function startScanning() {
    if (!(await QrScanner.hasCamera())) {
        console.log("Нет камеры");
        setError("Ошибка");
        restartButton.style.display = 'flex';

        return;
    }
    setError("");
    restartButton.style.display = 'none';

    try {
        await scanner.start();
    }
    catch (err) {
        scanner.destroy();
        setError("Ошибка");
        restartButton.style.display = 'flex';

        console.log(err);
    }
}

document.addEventListener('DOMContentLoaded', startScanning);

restartButton.addEventListener('click', () => {
    startScanning();
    clearTable();
});

fileInput.addEventListener('change', async () => {
    const file = fileInput.files[0];
    if (!file) {
        return;
    }

    try {
        const result = await QrScanner.scanImage(file, { returnDetailedScanResult: true });

        stopScanning(scanner);
        setResult(result.data);
    }
    catch (err) {
        console.log(`Ошибка при сканировании файла: ${err}`);
    }
});

cancelButton.addEventListener('click', () => { scanner.stop(); restartButton.style.display = 'flex'; });