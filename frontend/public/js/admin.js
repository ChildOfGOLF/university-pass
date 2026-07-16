import { apiMethods } from "./api";

const form = document.getElementById("admin-login-form");
const email = document.getElementById("admin-email");
const password = document.getElementById("admin-password");
const submitButton = document.getElementById("admin-submit-button");
const errorElement = document.getElementById("admin-error");

function showError(message) {
    errorElement.textContent = message;
}

function clearError() {
    errorElement.textContent = "";
}

form.addEventListener("submit", async(event) => {
    event.preventDefault();
    clearError();

    const email = email.value.trim();
    const password = password.value.trim();

    if (!email || !password) {
        showError("Заполните пжшка всее поля");
        return;
    }

    try {
        const {token} = await apiMethods.adminLogin({
            email,
            password
        })

        if (!token) {
            throw new Error ("Сервер не вернул токе");
        }

        localStorage.setItem("admin_token", token);

    }
    catch (err) {
        showError(err.message || "Не вошли");
    }
    finally {
        
    }
})