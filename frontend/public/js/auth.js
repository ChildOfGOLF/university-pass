//Логика для auth
import {apiMethods} from './api.js';

/* if (localStorage.getItem('secret_key')) {
    window.location.href = 'pass.html';
}*/

const form = document.getElementById('login-form');
const loginInput = document.getElementById('email');
const passwordInput = document.getElementById('password');
const submitButton = document.getElementById('submit-button');
const toggleButton = document.getElementById("toggle-button");
const errorElement = document.getElementById('Error');
const remembrMe = document.getElementById('rememberMe');

function getDeviceId() {
    let deviceId = localStorage.getItem('device_id');
    if (!deviceId) {
        deviceId = crypto.randomUUID();
        localStorage.setItem('device_id', deviceId);
    }
    return deviceId;
}

function setLoading(isLoading) {
    submitButton.disabled = isLoading;
    submitButton.textContent = isLoading ? "Загружаем" : "Войти";
}

function showError(message) {
    errorElement.textContent = message;
}

function clearError() {
    errorElement.textContent = "";
}

toggleButton.addEventListener('click', () => {
    let passwordCheck = passwordInput.type == "password";
    passwordInput.type = passwordCheck ? 'text' : 'password';
    toggleButton.classList.toggle('button-show--active', passwordCheck);
    toggleButton.setAttribute('aria-label', passwordCheck ? 'Скрыть пароль' : 'Показать пароль');
});

const savedLogin = localStorage.getItem('saved_login');
if (savedLogin) {
    loginInput.value = savedLogin;
    remembrMe.checked = true;
}




form.addEventListener('submit', async(event) => {
    event.preventDefault();
    clearError();

    const login = loginInput.value.trim();
    const password = passwordInput.value;

    if (!login || !password) {
        showError("Заполните поля плиз");
        return;
    }
    setLoading(true);

    try {
        const {secret_key} = await apiMethods.login({
            email: login,
            password,
            device_id: getDeviceId()
        });

        if (!secret_key) {
            throw new Error('Не вернул сикрит ки');
        }

        if (remembrMe.checked) {
            localStorage.setItem('saved_login', login);
        }
        else {
            localStorage.removeItem('saved_login');
        }

        localStorage.setItem('secret_key', secret_key);
        setTimeout(()=>{
            window.location.href = 'pass.html'
        }, 1000);
    }
    catch (err) {
        showError("Не удается войти");
        console.error(err);
    } finally {
        setLoading(false);
    }
})

