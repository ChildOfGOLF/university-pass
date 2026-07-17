//Обертка для всех эндпойнтова
const BASE_URL = "http://localhost:8081";

async function request(path, options = {}) {
    const res = await fetch(BASE_URL + path, {
        headers: {
            'Content-Type': 'application/json', ...options.headers
        },
        ...options
    });
    if (!res.ok) {
        let errorMessage = await res.json().catch(() => ({}) );
        const message = errorMessage.message || errorMessage.error || errorMessage.detail || `Ошибка ${res.status}`
        throw new Error(message);
    } 
    return res.json();
}

export const apiMethods = {
    login: (data) => request('/auth/login', {method: "POST", body: JSON.stringify(data)}),
    verify: (data) => request('/scan/verify', {method: "POST", headers: {'X-Scanner-Key': 'test_api'}, body: JSON.stringify(data)}),

    //админские темы
    adminLogin: (data) => request('/admin/auth/login', {method: "POST", body: JSON.stringify(data)}),
    //работа с пользователями
    getUsers: (adminToken) => request('/admin/users', {headers: {'Authorization':`Bearer ${adminToken}`}}),
    createUser: (adminToken, data) => request('/admin/users', {method: "POST", body: JSON.stringify(data), headers: {'Authorization':`Bearer ${adminToken}`}}),
    deleteUser: (adminToken, id) => request(`/admin/users/${id}`, {method: "DELETE", headers: {'Authorization':`Bearer ${adminToken}`}}),
    updateUser: (adminToken, data, id) => request(`/admin/users/${id}`, {method: "PATCH", body: JSON.stringify(data), headers: {'Authorization':`Bearer ${adminToken}`}}),

    //работа с гостями
    getGuests: (adminToken) => request('/admin/guests', {headers: {'Authorization':`Bearer ${adminToken}`}}),
    createPass: (adminToken, data) => request('/admin/guests', {method: "POST", body: JSON.stringify(data), headers: {'Authorization':`Bearer ${adminToken}`}}),
    revokePass: (adminToken, id) => request(`/admin/guests/${id}/revoke`, {method: "POST", headers: {'Authorization':`Bearer ${adminToken}`}}),
};