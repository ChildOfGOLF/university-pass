import { apiMethods } from "./api.js";

const loginSection = document.getElementById("login-section")
const form = document.getElementById("admin-login-form");
const emailInput = document.getElementById("admin-email");
const passwordInput = document.getElementById("admin-password");
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

    const email = emailInput.value.trim();
    const password = passwordInput.value.trim();

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
        showAdminPanel();
        loadUsers();
    }
    catch (err) {
        showError(err.message || "Не вошли");
    }
    finally {
        
    }
})

//Начинаем работать с таблицами
const adminPanel = document.getElementById("admin-panel");
const userTableBody = document.getElementById("users-table-body");
const guestTableBody = document.getElementById("guests-table-body");
const logoutBetton = document.getElementById("logout-admin");
const createUserButton = document.getElementById("create-user-button");
const tabButtons = document.querySelectorAll(".tab-button");
const tabContents = document.querySelectorAll(".tab-content");

function showAdminPanel () {
    loginSection.classList.add("visually-hidden");
    adminPanel.classList.remove("visually-hidden");
}

tabButtons.forEach(button => {
    button.addEventListener("click", () => {
        const tabName = button.dataset.tab;

        tabButtons.forEach(btn => btn.classList.remove("active"));
        tabContents.forEach(cnt => cnt.classList.add("visually-hidden"));

        button.classList.add("active");
        document.getElementById(`${tabName}-tab`).classList.remove("visually-hidden");

        if (tabName == "guests") {
            loadGuests();
        }
    });
})

async function loadUsers() {
    const token = localStorage.getItem("admin_token");
    try {
        const users = await apiMethods.getUsers(token);

        if (!Array.isArray(users)) {
            userTableBody.innerHTML = "<tr> <td colspan = '7'>Пользователи не нашлись</td></tr>";
            return;
        }

        userTableBody.innerHTML = users.map(user=>
            `<tr>
                <td> ${user.id} </td>
                <td> ${user.last_name} ${user.first_name} ${user.patronymic}</td>
                <td> ${user.email} </td>
                <td> ${new Date(user.created_at).toLocaleDateString('ru-RU')}</td>
                <td> ${user.is_active ? 'активен' : 'неактивен'} </td>
                <td> ${user.role} </td>
                <td>
                    <button class = "delete-button" onclick = "deleteUser(${user.id}")>пока пупсик</button>
                </td>`
        ).join();
    }
    catch (err) {
        userTableBody.innerHTML = `<tr><ts colspan = '7' style = "color: red">Ошибка ${err.message}</td></tr>`
    }
}

async function loadGuests() {
    const token = localStorage.getItem("admin_token");
    try {
        const guest = await apiMethods.getGuests(token);


    }
}
