import './style.css'
import '@material/web/button/filled-button.js';
import '@material/web/button/outlined-button.js';
import '@material/web/checkbox/checkbox.js';
import '@material/web/icon/icon.js';
import '@material/web/iconbutton/icon-button.js';
import '@material/web/textfield/outlined-text-field.js';
import '@material/web/labs/card/elevated-card.js';
import '@material/web/list/list.js';
import '@material/web/list/list-item.js';
import '@material/web/fab/fab.js';
import { MdOutlinedTextField } from '@material/web/textfield/outlined-text-field.js';
import { login, getMods, deleteMod, getModVersions, deleteVersion } from './api';
import { isLoggedIn, setSession, logout, getUser } from './auth';
import { showModDialog } from './mod-dialog';
import { showCreateVersionDialog } from './version-dialog';



const app = document.querySelector<HTMLDivElement>('#app')!;



function renderLogin() {
    app.innerHTML = `
    <div style="display: flex; justify-content: center; align-items: center; height: 100vh;">
      <md-elevated-card style="padding: 24px; min-width: 300px; display: flex; flex-direction: column; gap: 16px;">
        <h2 style="margin: 0; text-align: center;">Login</h2>
        <md-outlined-text-field label="Username" id="username" type="text"></md-outlined-text-field>
        <md-outlined-text-field label="Password" id="password" type="password"></md-outlined-text-field>
        <div id="error-msg" style="color: red; font-size: 0.9em; display: none;"></div>
        <md-filled-button id="login-btn">Login</md-filled-button>
      </md-elevated-card>
    </div>
  `;

    const usernameInput = document.getElementById('username') as MdOutlinedTextField;
    const passwordInput = document.getElementById('password') as MdOutlinedTextField;
    const loginBtn = document.getElementById('login-btn')!;
    const errorMsg = document.getElementById('error-msg')!;

    loginBtn.addEventListener('click', async () => {
        errorMsg.style.display = 'none';
        try {
            const resp = await login(usernameInput.value, passwordInput.value);
            setSession(resp.token, resp.user);
            renderDashboard();
        } catch (e: any) {
            errorMsg.textContent = e.message;
            errorMsg.style.display = 'block';
        }
    });
}

function renderDashboard() {
    const user = getUser();
    app.innerHTML = `
  <div class="app-bar">
    <md-icon-button>
        <md-icon>menu</md-icon>
    </md-icon-button>
    <span class="title">Au Mod Installer Admin</span>
    <div style="flex: 1;"></div>
    <span style="margin-right: 16px;">${user?.username || 'User'}</span>
    <md-icon-button id="logout-btn">
        <md-icon>logout</md-icon>
    </md-icon-button>
  </div>
    <div class="main-content">
     <md-elevated-card style="padding: 16px; margin: 16px;">
        <div style="display: flex; justify-content: space-between; align-items: center;">
            <h2 style="margin: 0;">Mods Repository</h2>
            <md-fab id="create-mod-btn" size="small">
                <md-icon slot="icon">add</md-icon>
            </md-fab>
        </div>
        <div id="mods-list" style="margin-top: 16px;">Loading mods...</div>
     </md-elevated-card>
  </div>
`;

    document.getElementById('logout-btn')!.addEventListener('click', () => {
        logout();
    });

        document.getElementById('create-mod-btn')!.addEventListener('click', () => {
        showModDialog(() => {
            loadMods(); // Reload list on success
        });
    });


    loadMods();
}


async function loadMods() {
    const modsListEl = document.getElementById('mods-list');
    if (!modsListEl) return;

    try {
        const mods = await getMods();
        if (mods.length === 0) {
            modsListEl.innerHTML = '<p>No mods found.</p>';
            return;
        }

        let html = '<md-list>';
        mods.forEach((mod: any) => {
            html += `
            <md-list-item>
                <div slot="headline">${mod.name}</div>
                <div slot="supporting-text">${mod.description || 'No description'}</div>
                <div slot="trailing-supporting-text">${mod.author}</div>
                <div slot="end" style="display: flex; gap: 8px;">
                    <md-icon-button class="view-versions-btn" data-id="${mod.id}" title="View Versions">
                        <md-icon>list</md-icon>
                    </md-icon-button>
                    <md-icon-button class="edit-mod-btn" data-id="${mod.id}" title="Edit Mod">
                        <md-icon>edit</md-icon>
                    </md-icon-button>
                    <md-icon-button class="delete-mod-btn" data-id="${mod.id}" style="--md-icon-button-icon-color: red;" title="Delete Mod">
                        <md-icon>delete</md-icon>
                    </md-icon-button>
                </div>
            </md-list-item>
            <div id="versions-container-${mod.id}" style="display: none; padding-left: 32px; background-color: #1a1a1a;">
                <div style="display: flex; justify-content: space-between; align-items: center; padding: 8px;">
                    <h3 style="margin: 0; font-size: 1rem;">Versions</h3>
                    <md-filled-button class="upload-version-btn" data-id="${mod.id}" style="--md-filled-button-container-height: 32px;">Upload Version</md-filled-button>
                </div>
                <div class="versions-list" data-id="${mod.id}">Loading versions...</div>
            </div>
            <div style="height: 1px; background-color: #333;"></div>
            `;
        });
        html += '</md-list>';
        modsListEl.innerHTML = html;

        // Add event listeners
        modsListEl.querySelectorAll('.view-versions-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                const modID = btn.getAttribute('data-id')!;
                const container = document.getElementById(`versions-container-${modID}`)!;
                if (container.style.display === 'none') {
                    container.style.display = 'block';
                    loadVersions(modID);
                } else {
                    container.style.display = 'none';
                }
            });
        });

        modsListEl.querySelectorAll('.upload-version-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                const modID = btn.getAttribute('data-id')!;
                showCreateVersionDialog(modID, () => loadVersions(modID));
            });
        });

        modsListEl.querySelectorAll('.edit-mod-btn').forEach(btn => {
            btn.addEventListener('click', () => {
                const modID = btn.getAttribute('data-id');
                const mod = mods.find((m: any) => m.id === modID);
                showModDialog(() => loadMods(), mod);
            });
        });

        modsListEl.querySelectorAll('.delete-mod-btn').forEach(btn => {
            btn.addEventListener('click', async () => {
                const modID = btn.getAttribute('data-id');
                if (confirm(`Are you sure you want to delete mod ${modID}?`)) {
                    try {
                        await deleteMod(modID!);
                        loadMods();
                    } catch (e: any) {
                        alert(e.message);
                    }
                }
            });
        });

    } catch (e: any) {
        modsListEl.innerHTML = `<p style="color: red;">Error loading mods: ${e.message}</p>`;
    }
}

async function loadVersions(modID: string) {
    const container = document.querySelector(`.versions-list[data-id="${modID}"]`)!;
    try {
        const versions = await getModVersions(modID);
        if (versions.length === 0) {
            container.innerHTML = '<p style="padding: 8px;">No versions found.</p>';
            return;
        }

        let html = '<md-list>';
        versions.forEach((v: any) => {
            html += `
            <md-list-item>
                <div slot="headline">${v.id}</div>
                <div slot="supporting-text">Created: ${new Date(v.created_at).toLocaleString()}</div>
                <div slot="end">
                    <md-icon-button class="delete-version-btn" data-mod-id="${modID}" data-ver-id="${v.id}" style="--md-icon-button-icon-color: red;">
                        <md-icon>delete</md-icon>
                    </md-icon-button>
                </div>
            </md-list-item>
            `;
        });
        html += '</md-list>';
        container.innerHTML = html;

        container.querySelectorAll('.delete-version-btn').forEach(btn => {
            btn.addEventListener('click', async () => {
                const mid = btn.getAttribute('data-mod-id')!;
                const vid = btn.getAttribute('data-ver-id')!;
                if (confirm(`Are you sure you want to delete version ${vid}?`)) {
                    try {
                        await deleteVersion(mid, vid);
                        loadVersions(mid);
                    } catch (e: any) {
                        alert(e.message);
                    }
                }
            });
        });
    } catch (e: any) {
        container.innerHTML = `<p style="color: red; padding: 8px;">Error loading versions: ${e.message}</p>`;
    }
}




if (isLoggedIn()) {
    renderDashboard();
} else {
    renderLogin();
}


