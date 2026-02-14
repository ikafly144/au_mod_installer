import { getToken } from './auth';

export const API_BASE = 'http://localhost:8180/api/v1';

export async function exchangeDiscordCode(code: string): Promise<{ token: string, user: any }> {
    const response = await fetch(`${API_BASE}/auth/discord/callback?code=${encodeURIComponent(code)}`);

    const text = await response.text();
    if (!response.ok) {
        try {
            const error = JSON.parse(text);
            throw new Error(error.error || 'Discord login failed');
        } catch (e) {
            if (e instanceof Error && e.message !== 'Discord login failed') {
                throw new Error(`Discord login failed: ${text || response.statusText}`);
            }
            throw e;
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

    if (!response.ok) {
        const text = await response.text();
        try {
            const error = JSON.parse(text);
            throw new Error(error.error || 'Failed to fetch mods');
        } catch (e) {
            throw new Error(`Failed to fetch mods: ${text || response.statusText}`);
        }
    }

    return await response.json();
}

export async function getMod(modID: string): Promise<any> {
    const token = getToken();
    const headers: any = {};
    if (token) {
        headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${API_BASE}/mods/${modID}`, {
        headers: headers
    });

    if (!response.ok) {
        const text = await response.text();
        try {
            const error = JSON.parse(text);
            throw new Error(error.error || 'Failed to fetch mod');
        } catch (e) {
            throw new Error(`Failed to fetch mod: ${text || response.statusText}`);
        }
    }

    return await response.json();
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

    if (!response.ok) {
        const text = await response.text();
        try {
            const error = JSON.parse(text);
            throw new Error(error.error || 'Failed to create mod');
        } catch (e) {
            throw new Error(`Failed to create mod: ${text || response.statusText}`);
        }
    }

    return await response.json();
}

export async function updateMod(modID: string, mod: any): Promise<any> {
    const token = getToken();
    const headers: any = { 'Content-Type': 'application/json' };
    if (token) {
        headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${API_BASE}/mods/${modID}`, {
        method: 'PUT',
        headers: headers,
        body: JSON.stringify(mod)
    });

    if (!response.ok) {
        const text = await response.text();
        try {
            const error = JSON.parse(text);
            throw new Error(error.error || 'Failed to update mod');
        } catch (e) {
            throw new Error(`Failed to update mod: ${text || response.statusText}`);
        }
    }

    return await response.json();
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

export async function getModVersions(modID: string): Promise<any[]> {
    const token = getToken();
    const headers: any = {};
    if (token) {
        headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${API_BASE}/mods/${modID}/versions`, {
        headers: headers
    });

    if (!response.ok) {
        const text = await response.text();
        try {
            const error = JSON.parse(text);
            throw new Error(error.error || 'Failed to fetch versions');
        } catch (e) {
            throw new Error(`Failed to fetch versions: ${text || response.statusText}`);
        }
    }

    return await response.json();
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

    if (!response.ok) {
        const text = await response.text();
        try {
            const error = JSON.parse(text);
            throw new Error(error.error || 'Failed to create version');
        } catch (e) {
            throw new Error(`Failed to create version: ${text || response.statusText}`);
        }
    }

    return await response.json();
}

export async function updateVersion(modID: string, versionID: string, version: any): Promise<any> {
    const token = getToken();
    const headers: any = { 'Content-Type': 'application/json' };
    if (token) {
        headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${API_BASE}/mods/${modID}/versions/${versionID}`, {
        method: 'PUT',
        headers: headers,
        body: JSON.stringify(version)
    });

    if (!response.ok) {
        const text = await response.text();
        try {
            const error = JSON.parse(text);
            throw new Error(error.error || 'Failed to update version');
        } catch (e) {
            throw new Error(`Failed to update version: ${text || response.statusText}`);
        }
    }

    return await response.json();
}

export async function deleteVersion(modID: string, versionID: string): Promise<void> {
    const token = getToken();
    const headers: any = {};
    if (token) {
        headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${API_BASE}/mods/${modID}/versions/${versionID}`, {
        method: 'DELETE',
        headers: headers
    });

    if (!response.ok) {
        const text = await response.text();
        try {
            const error = JSON.parse(text);
            throw new Error(error.error || 'Failed to delete version');
        } catch (e) {
            throw new Error(`Failed to delete version: ${text || response.statusText}`);
        }
    }
}

export async function getGitHubReleases(modID: string): Promise<any[]> {
    const token = getToken();
    const headers: any = {};
    if (token) {
        headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${API_BASE}/mods/${modID}/github/releases`, {
        headers: headers
    });

    if (!response.ok) {
        const text = await response.text();
        try {
            const error = JSON.parse(text);
            throw new Error(error.error || 'Failed to fetch GitHub releases');
        } catch (e) {
            throw new Error(`Failed to fetch GitHub releases: ${text || response.statusText}`);
        }
    }

    return await response.json();
}

export async function createVersionFromGitHub(modID: string, tag: string): Promise<any> {
    const token = getToken();
    const headers: any = { 'Content-Type': 'application/json' };
    if (token) {
        headers['Authorization'] = `Bearer ${token}`;
    }

    const response = await fetch(`${API_BASE}/mods/${modID}/versions/from-github`, {
        method: 'POST',
        headers: headers,
        body: JSON.stringify({ tag })
    });

    if (!response.ok) {
        const text = await response.text();
        try {
            const error = JSON.parse(text);
            throw new Error(error.error || 'Failed to create version from GitHub');
        } catch (e) {
            throw new Error(`Failed to create version from GitHub: ${text || response.statusText}`);
        }
    }

    return await response.json();
}
