//Обертка для всех эндпойнтов
const BASE_URL = /* "https://localhost:8080" */ "/api";

async function request(path, options = {}) {
    const res = await fetch(BASE_URL + path, {
        headers: {
            'Content-Type': 'application/json', ...options.headers
        },
        credentials: 'include',
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
    verify: (data) => request('/scan/verify', {method: "POST", headers: {'X-Scanner-Key': 'test_api'}, body: JSON.stringify(data)})
}