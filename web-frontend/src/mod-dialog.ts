import { createMod } from './api';
import '@material/web/dialog/dialog.js';
import '@material/web/button/text-button.js';
import '@material/web/button/filled-button.js';
import '@material/web/textfield/outlined-text-field.js';

export function showCreateModDialog(onSuccess: () => void) {
    const dialogId = 'create-mod-dialog-' + Date.now();
    const dialogHtml = `
    <md-dialog id="${dialogId}">
        <div slot="headline">Create New Mod</div>
        <form slot="content" id="create-mod-form" method="dialog">
            <div style="display: flex; flex-direction: column; gap: 16px; padding-top: 10px;">
                <md-outlined-text-field label="ID" id="mod-id" required></md-outlined-text-field>
                <md-outlined-text-field label="Name" id="mod-name" required></md-outlined-text-field>
                <md-outlined-text-field label="Author" id="mod-author" required></md-outlined-text-field>
                <md-outlined-text-field label="Description" id="mod-desc"></md-outlined-text-field>
                <md-outlined-text-field label="Website" id="mod-website"></md-outlined-text-field>
                <md-outlined-text-field label="Type" id="mod-type" value="mod" supporting-text="mod, library, etc."></md-outlined-text-field>
            </div>
        </form>
        <div slot="actions">
            <md-text-button form="create-mod-form" value="cancel" onclick="this.closest('md-dialog').close()">Cancel</md-text-button>
            <md-filled-button id="create-mod-submit">Create</md-filled-button>
        </div>
    </md-dialog>
    `;

    document.body.insertAdjacentHTML('beforeend', dialogHtml);
    const dialog = document.getElementById(dialogId) as any;
    const submitBtn = dialog.querySelector('#create-mod-submit') as HTMLElement;

    dialog.show();

    submitBtn.addEventListener('click', async (e) => {
        e.preventDefault();
        const id = (dialog.querySelector('#mod-id') as any).value;
        const name = (dialog.querySelector('#mod-name') as any).value;
        const author = (dialog.querySelector('#mod-author') as any).value;
        const description = (dialog.querySelector('#mod-desc') as any).value;
        const website = (dialog.querySelector('#mod-website') as any).value;
        const type = (dialog.querySelector('#mod-type') as any).value;

        if (!id || !name || !author) {
            alert("ID, Name, and Author are required.");
            return;
        }

        try {
            await createMod({
                id,
                name,
                author,
                description,
                website,
                type,
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString()
            });
            dialog.close();
            onSuccess();
        } catch (e: any) {
            alert(e.message);
        }
    });

    dialog.addEventListener('closed', () => {
        dialog.remove();
    });
}
