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

let userBuff = [];

async function loadUsers() {
    const token = localStorage.getItem("admin_token");
    
    try {
        const users = await apiMethods.getUsers(token);
        userBuff = users;

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
                    <button class = "user-button" onclick = "openUpdateUser('${user.id}')">обновить</button>
                </td>
                <td>
                    <button class = "user-button" onclick = "deleteUser('${user.id}')">удалить</button>
                </td>
            </tr>`
        ).join('');
    }
    catch (err) {
        userTableBody.innerHTML = `<tr><td colspan = '7' style = "color: red">Ошибка ${err.message}</td></tr>`
    }
}

async function checkAuthAndInit() {
    const savedToken = localStorage.getItem("admin_token");
    if (!savedToken) return;

    try {
        await apiMethods.getUsers(savedToken);
        showAdminPanel();
        loadUsers();
    } catch (err) {

        localStorage.removeItem("admin_token");
    }
}

checkAuthAndInit();

//Удаление юзера
window.deleteUser = async (userId) => {
    const token = localStorage.getItem("admin_token");
    try {
        await apiMethods.deleteUser(token, userId);
        loadUsers()
    } catch (err) {
        console.log("Ошибка при удалении");
    }
}

//Форма обновления юзера

let editingUserId = 0;

window.openUpdateUser = (userId) => {
    const user = userBuff.find(u => u.id == userId);
    if (!user) {
        return;
    }

    editingUserId = userId;

    document.getElementById("update-user-second_name").value = user.last_name;
    document.getElementById("update-user-first_name").value = user.first_name;
    document.getElementById("update-user-patronymic").value = user.patronymic;
    document.getElementById("update-user-phone").value = user.phone;

    document.getElementById("modal-update-user").showPopover();
}


const formUpdateUser = document.getElementById("form-update-user");

if (formUpdateUser) {
    formUpdateUser.addEventListener("submit", async(event) => {
        event.preventDefault();

        const token = localStorage.getItem("admin_token");
        const data = {
            first_name: document.getElementById("update-user-first_name").value.trim(),
            last_name: document.getElementById("update-user-second_name").value.trim(),
            patronymic: document.getElementById("update-user-patronymic").value.trim(),
            phone: document.getElementById("update-user-phone").value.trim()
        };

        try {
            await apiMethods.updateUser(token, data, editingUserId);

            document.getElementById("modal-update-user").hidePopover();
            loadUsers();
        }
        catch (err) {
            console.log(err.message);
            alert(err.message);
        }
    })
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
                <td>${guest.is_used ? "Да" : "Нет"}</td>
                <td>${guest.is_entered ? "Да" : "Нет"}</td>
                <td>
                    <button class = "delete-button" onclick = "deleteGuest('${guest.id}')">пока пупс</button>
                </td>
            </tr>
        `).join('');
    }
    catch(err) {
         guestTableBody.innerHTML = `<tr><td colspan = '7' style = "color: red">Ошибка ${err.message}</td></tr>`;
    }
}



window.deleteGuest = async (guestId) => {
    const token = localStorage.getItem("admin_token");
    try {
        await apiMethods.revokePass(token, guestId);
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
        const groupName = document.getElementById("user-group").value.trim();

        
    
        try {
            const groups = await apiMethods.getGroups(token);
            const group = groups.find(g=> g.name == groupName);

            if (!group) {
                alert("группы не найдено");
                return;
            }

            const data = {
                email: document.getElementById("user-email").value.trim(),
                first_name: document.getElementById("user-first_name").value.trim(),
                group_id: group.id,
                last_name: document.getElementById("user-second_name").value.trim(),
                password: document.getElementById("user-password").value.trim(),
                patronymic: document.getElementById("user-patronymic").value.trim(),
                phone: document.getElementById("user-phone").value.trim(),
                role: document.getElementById("user-role").value.trim()
            };
            const res = await apiMethods.createUser(token, data);

            document.getElementById("modal-create-user").hidePopover();
            formCreateUser.reset();
            loadUsers();
        } catch (err) {
            console.log(err.message);
            alert(err.message)
        }
    })
}

if (formCreateGuest) {
    formCreateGuest.addEventListener("submit", async (event) => {
        event.preventDefault();

        const token = localStorage.getItem("admin_token");

        try {
            const data = {
                first_name: document.getElementById("guest-first_name").value.trim(),
                last_name: document.getElementById("guest-second_name").value.trim(),
                patronymic: document.getElementById("guest-patronymic").value.trim(),
                purpose: document.getElementById("guest-purpose").value.trim(),
                valid_from: new Date(document.getElementById("guest-valid_from").value).toISOString(),
                valid_to: new Date(document.getElementById("guest-valid_to").value).toISOString()
            }

            await apiMethods.createPass(token, data);

            document.getElementById("modal-create-guest").hidePopover();
            formCreateGuest.reset();
            loadGuests();
        }
        catch (err) {
            console.log(err.message);
            alert(err.message);
        }
    })
}

//логи
const logsButton = document.getElementById("logs-button");


async function loadLogs() {
    const token = localStorage.getItem("admin_token");
    const logsContainer = document.getElementById("logs-container");


    try {
        const logs = await apiMethods.getLogs(token);

        if (!Array.isArray(logs)) {
            logsContainer.innerHTML="<p class = 'logs-error' style = 'color:red'>Логи не найдены</p>";
            return;
        }

        logs.sort((a, b) => new Date(b.logged_at) - new Date(a.logged_at));

        logsContainer.innerHTML = logs.map(log => {
            const logDirection = log.direction == "enter" ? "Вход" : "Выход";
            const logType = log.direction;

            return `
                <div class = "log-item small-text">
                    <div class = "log-header ${logType}">
                        <span class = "log-direction"> ${logDirection} </span>
                        <span class = "log-date">${log.logged_at} </span>
                        <span class = "log-allowed">Пропущен: ${log.is_allowed ? "Да" : "Нет"} </span>
                        <span class = "logs-building">Строение: ${log.building}</span>
                        <span class = "log-gate">КПП: ${log.gate}</span>
                        <span class = "log-acces_point">Точка доступа: ${log.acces_point}</span>
                    </div>
                    <span class = "log-name">${log.full_name} - ${log.person_type}</span>
                    <span class = "log-reason">${log.reason}</span>
                </div>
            `
        }).join('');
    }
    catch(err) {
        console.log(err.message);
    }
}

if (logsButton) {
    logsButton.addEventListener("click", () => {
        loadLogs();
    });
}