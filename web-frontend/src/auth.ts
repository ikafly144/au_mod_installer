const TOKEN_KEY = 'auth_token';
const USER_KEY = 'auth_user';

export function setSession(token: string, user: any) {
    localStorage.setItem(TOKEN_KEY, token);
    localStorage.setItem(USER_KEY, JSON.stringify(user));
}

export function getToken(): string | null {
    return localStorage.getItem(TOKEN_KEY);
}

export function getUser(): any | null {
    const u = localStorage.getItem(USER_KEY);
    if (!u || u === 'undefined') return null;
    try {
        return JSON.parse(u);
    } catch (e) {
        console.error("Failed to parse user from local storage", e);
        return null;
    }
}


export function logout() {
    localStorage.removeItem(TOKEN_KEY);
    localStorage.removeItem(USER_KEY);
    window.location.reload();
}

export function isLoggedIn(): boolean {
    return !!getToken();
}
