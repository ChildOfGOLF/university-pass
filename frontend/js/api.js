//Обертка для всех эндпойнтов
const BASE_URL = "/...";

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
    
}