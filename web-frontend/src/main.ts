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
import { MdOutlinedTextField } from '@material/web/textfield/outlined-text-field.js';
import { login, getMods } from './api';
import { isLoggedIn, setSession, logout, getUser } from './auth';

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
        <h2>Mods Repository</h2>
        <div id="mods-list">Loading mods...</div>
     </md-elevated-card>
  </div>
`;

    document.getElementById('logout-btn')!.addEventListener('click', () => {
        logout();
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
            </md-list-item>
            <div style="height: 1px; background-color: #333;"></div>
            `;
        });
        html += '</md-list>';
        modsListEl.innerHTML = html;
    } catch (e: any) {
        modsListEl.innerHTML = `<p style="color: red;">Error loading mods: ${e.message}</p>`;
    }
}


if (isLoggedIn()) {
    renderDashboard();
} else {
    renderLogin();
}


