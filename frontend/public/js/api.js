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
        throw new Error(errorMessage.message);
    } 
    return res.json();
}

export const apiMethods = {
    login: (data) => request('/auth/login', {method: "POST", body: JSON.stringify(data)}),
    verify: (data) => request('/scan/verify', {method: "POST", headers: {'X-Scanner-Key': 'test_api'}, body: JSON.stringify(data)}),
    adminLogin: (data) => request('/admin/auth/login', {method: "POST", body: JSON.stringify(data)}),
}