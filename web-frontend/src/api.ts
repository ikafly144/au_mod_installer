import { getToken } from './auth';

export const API_BASE = 'http://localhost:8180/api/v1';

export async function login(username: string, password: string): Promise<{ token: string, user: any }> {
    const response = await fetch(`${API_BASE}/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password })
    });

    if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Login failed');
    }

    return response.json();
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

    if (!response.ok) {
        const error = await response.json();
        throw new Error(error.error || 'Failed to fetch mods');
    }

    return response.json();
}

