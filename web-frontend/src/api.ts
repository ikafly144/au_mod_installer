import { getToken } from './auth';

export const API_BASE = 'http://localhost:8180/api/v1';

export async function login(username: string, password: string): Promise<{ token: string, user: any }> {
        const response = await fetch(`${API_BASE}/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password })
    });

    const text = await response.text();
    if (!response.ok) {
        try {
            const error = JSON.parse(text);
            throw new Error(error.error || 'Login failed');
        } catch (e) {
             throw new Error(`Login failed: ${text || response.statusText}`);
        }
    }

    try {
        return JSON.parse(text);
    } catch (e) {
        throw new Error(`Invalid server response: ${text}`);
    }
}

export async function getMods(): Promise<any[]> {
    const token = getToken();
    const headers: any = {};
    if (token) {
        headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${API_BASE}/mods`, {
        headers: headers
    });

    const text = await response.text();
    if (!response.ok) {
        try {
            const error = JSON.parse(text);
             throw new Error(error.error || 'Failed to fetch mods');
        } catch (e) {
            throw new Error(`Failed to fetch mods: ${text || response.statusText}`);
        }
    }

    try {
        return JSON.parse(text);
    } catch (e) {
         throw new Error(`Invalid server response: ${text}`);
    }
}


