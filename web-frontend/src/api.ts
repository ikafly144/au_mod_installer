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

export async function createMod(mod: any): Promise<any> {
    const token = getToken();
    const headers: any = { 'Content-Type': 'application/json' };
    if (token) {
        headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${API_BASE}/mods`, {
        method: 'POST',
        headers: headers,
        body: JSON.stringify(mod)
    });

    const text = await response.text();
    if (!response.ok) {
        try {
            const error = JSON.parse(text);
            throw new Error(error.error || 'Failed to create mod');
        } catch (e) {
            throw new Error(`Failed to create mod: ${text || response.statusText}`);
        }
    }

    try {
        return JSON.parse(text);
    } catch (e) {
        throw new Error(`Invalid server response: ${text}`);
    }
}

export async function uploadFile(file: File): Promise<string> {
    const token = getToken();
    const headers: any = {};
    if (token) {
        headers['Authorization'] = `Bearer ${token}`;
    }

    const formData = new FormData();
    formData.append('file', file);

    const response = await fetch(`${API_BASE}/upload`, {
        method: 'POST',
        headers: headers,
        body: formData
    });

    if (!response.ok) {
        const text = await response.text();
        try {
            const error = JSON.parse(text);
            throw new Error(error.error || 'Upload failed');
        } catch (e) {
            throw new Error(`Upload failed: ${text || response.statusText}`);
        }
    }

    const data = await response.json();
    return data.url;
}

export async function createVersion(modID: string, version: any): Promise<any> {
    const token = getToken();
    const headers: any = { 'Content-Type': 'application/json' };
    if (token) {
        headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${API_BASE}/mods/${modID}/versions`, {
        method: 'POST',
        headers: headers,
        body: JSON.stringify(version)
    });

    const text = await response.text();
    if (!response.ok) {
        try {
            const error = JSON.parse(text);
            throw new Error(error.error || 'Failed to create version');
        } catch (e) {
            throw new Error(`Failed to create version: ${text || response.statusText}`);
        }
    }

    try {
        return JSON.parse(text);
    } catch (e) {
        throw new Error(`Invalid server response: ${text}`);
    }
}


export async function deleteMod(modID: string): Promise<void> {

    const token = getToken();
    const headers: any = {};
    if (token) {
        headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${API_BASE}/mods/${modID}`, {
        method: 'DELETE',
        headers: headers
    });

    if (!response.ok) {
        const text = await response.text();
        try {
            const error = JSON.parse(text);
            throw new Error(error.error || 'Failed to delete mod');
        } catch (e) {
            throw new Error(`Failed to delete mod: ${text || response.statusText}`);
        }
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


