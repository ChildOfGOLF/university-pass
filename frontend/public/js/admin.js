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
const logoutButton = document.getElementById("logout-admin");
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
                <td>${user.id}</td>
                <td>${user.last_name} ${user.first_name} ${user.patronymic}</td>
                <td>${user.email} </td>
                <td>${new Date(user.created_at).toLocaleDateString('ru-RU')}</td>
                <td>${user.is_active ? 'активен' : 'неактивен'}</td>
                <td>${user.role}</td>
                <td>
                    <button class = "delete-button" onclick = "deleteUser(${user.id})">пока пупсик</button>
                </td>
            </tr>`
        ).join('');
    }
    catch (err) {
        userTableBody.innerHTML = `<tr><td colspan = '7' style = "color: red">Ошибка ${err.message}</td></tr>`
    }
}

async function loadGuests() {
    const token = localStorage.getItem("admin_token");
    try {
        const guests = await apiMethods.getGuests(token);

        if (!Array.isArray(guests)) {
            guestTableBody.innerHTML = "<tr> <td colspan ='8'>Не удалось найти пользователей</td></tr>";
            return;
        }

        guestTableBody.innerHTML = guests.map(guest => `
            <tr>
                <td>${guest.id}</td>
                <td>${guest.last_name} ${guest.first_name} ${guest.patronymic}</td>
                <td>${new Date(guest.created_at).toLocaleDateString('ru-RU')}</td>
                <td>${new Date(guest.valid_to).toLocaleDateString('ru-RU')}</td>
                <td>${guest.purpose}</td>
                <td>${guest.is_used}</td>
                <td>${guest.is_entered}</td>
                <td>
                    <button class = "delete-button" onclick = "deleteGuest(${guest.id})">пока пупс</button>
                </td>
            </tr>
        `).join('');
    }
    catch(err) {
         guestTableBody.innerHTML = `<tr><td colspan = '7' style = "color: red">Ошибка ${err.message}</td></tr>`;
    }
}

window.deleteUser = async (userId) => {
    const token = localStorage.getItem("admin_token");
    try {
        await apiMethods.deleteUser(token, userId);
        loadUsers()
    } catch (err) {
        console.log("Ошибка при удалении");
    }
}

window.deleteGuest = async (guestId) => {
    const token = localStorage.getItem("admin_token");
    try {
        await apiMethods.deleteGuest(token, guestId);
        loadGuests()
    } catch (err) {
        console.log("Ошибка при удалении");
    }
}

if (logoutButton) {
    logoutButton.addEventListener("click", () => {
        localStorage.removeItem("admin_token");
        window.location.reload();
    });
}

// Модальные окна для создание гостей и пользователей
const formCreateUser = document.getElementById("form-create-user");
const formCreateGuest = document.getElementById("form-create-guest");

if (formCreateUser) {
    formCreateUser.addEventListener("submit", async(event) => {
        event.preventDefault();
        
        const token = localStorage.getItem("admin_token");
        const data = {
            email: document.getElementById("user-email").value.trim(),
            first_name: document.getElementById("user-first_name").value.trim(),
            group_id: 0,
            last_name: document.getElementById("user-second_name").value.trim(),
            password: document.getElementById("user-password").value.trim(),
            patronymic: document.getElementById("user-patronymic").value.trim(),
            phone: document.getElementById("user-phone").value.trim(),
            role: document.getElementById("user-role").value.trim()
        };
        try {
            const res = await apiMethods.createUser(token, data);

            document.getElementById("form-create-user").hidePopover();
            formCreateUser.reset();
            loadUsers();
        } catch (res) {
            console.log(res);
        }
    })
}