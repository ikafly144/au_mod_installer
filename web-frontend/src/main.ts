import './style.css'
import '@material/web/button/filled-button.js';
import '@material/web/button/outlined-button.js';
import '@material/web/checkbox/checkbox.js';
import '@material/web/icon/icon.js';
import '@material/web/iconbutton/icon-button.js';
import '@material/web/textfield/outlined-text-field.js';
import '@material/web/labs/card/elevated-card.js';

document.querySelector<HTMLDivElement>('#app')!.innerHTML = `
  <div class="app-bar">
    <md-icon-button>
        <md-icon>menu</md-icon>
    </md-icon-button>
    <span class="title">Au Mod Installer Admin</span>
    <div style="flex: 1;"></div>
    <md-icon-button>
        <md-icon>account_circle</md-icon>
    </md-icon-button>
  </div>
  <div class="main-content">
     <md-elevated-card style="padding: 16px; margin: 16px;">
        <h2>Welcome</h2>
        <p>Material Web is set up.</p>
        <md-filled-button>Click Me</md-filled-button>
     </md-elevated-card>
  </div>
`

