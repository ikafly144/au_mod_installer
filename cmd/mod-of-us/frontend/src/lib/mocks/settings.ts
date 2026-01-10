// 設定の型定義
export interface AppSettings {
    general: {
        language: 'ja' | 'en';
        darkMode: boolean;
    };
    game: {
        installPath: string;
    };
}

// モックデータ
const mockSettings: AppSettings = {
    general: {
        language: 'ja',
        darkMode: false,
    },
    game: {
        installPath: 'C:\\Program Files (x86)\\Steam\\steamapps\\common\\Among Us',
    },
};

// 設定取得関数 (非同期を想定)
export async function getSettings(): Promise<AppSettings> {
    // 擬似的な遅延
    // await new Promise(resolve => setTimeout(resolve, 500));
    return { ...mockSettings };
}

// 設定保存関数
export async function saveSettings(settings: AppSettings): Promise<void> {
    console.log('Settings saved:', settings);
    // 実際の保存処理
}
