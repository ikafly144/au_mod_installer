import { createVersion, uploadFile } from './api';
import '@material/web/dialog/dialog.js';
import '@material/web/button/text-button.js';
import '@material/web/button/filled-button.js';
import '@material/web/textfield/outlined-text-field.js';

export function showCreateVersionDialog(modID: string, onSuccess: () => void) {
    const dialogId = 'create-ver-dialog-' + Date.now();
    const dialogHtml = `
    <md-dialog id="${dialogId}">
        <div slot="headline">Upload New Version</div>
        <form slot="content" id="create-ver-form" method="dialog">
            <div style="display: flex; flex-direction: column; gap: 16px; padding-top: 10px;">
                <md-outlined-text-field label="Version ID (e.g. v1.0.0)" id="ver-id" required></md-outlined-text-field>
                
                <div>
                    <label for="ver-file" style="display: block; margin-bottom: 8px; font-family: Roboto;">Mod File (.zip)</label>
                    <input type="file" id="ver-file" accept=".zip,.rar,.7z" required style="color: var(--md-sys-color-on-surface);">
                </div>
            </div>
        </form>
        <div slot="actions">
            <md-text-button form="create-ver-form" value="cancel" onclick="this.closest('md-dialog').close()">Cancel</md-text-button>
            <md-filled-button id="create-ver-submit">Upload & Create</md-filled-button>
        </div>
    </md-dialog>
    `;

    document.body.insertAdjacentHTML('beforeend', dialogHtml);
    const dialog = document.getElementById(dialogId) as any;
    const submitBtn = dialog.querySelector('#create-ver-submit') as HTMLElement;

    dialog.show();

    submitBtn.addEventListener('click', async (e) => {
        e.preventDefault();
        const verId = (dialog.querySelector('#ver-id') as any).value;
        const fileInput = dialog.querySelector('#ver-file') as HTMLInputElement;

        if (!verId) {
            alert("Version ID is required.");
            return;
        }
        if (!fileInput.files || fileInput.files.length === 0) {
            alert("File is required.");
            return;
        }

        const file = fileInput.files[0];
        
        // Show loading state
        submitBtn.innerText = "Uploading...";
        (submitBtn as any).disabled = true;

        try {
            const url = await uploadFile(file);
            
            await createVersion(modID, {
                id: verId,
                mod_id: modID,
                created_at: new Date().toISOString(),
                files: [
                    {
                        url: url,
                        file_type: "zip", // Assume zip for now
                        compatible: ["windows", "linux"] // Default compatibility
                    }
                ]
            });

            dialog.close();
            onSuccess();
        } catch (e: any) {
            alert(e.message);
            submitBtn.innerText = "Upload & Create";
            (submitBtn as any).disabled = false;
        }
    });

    dialog.addEventListener('closed', () => {
        dialog.remove();
    });
}
