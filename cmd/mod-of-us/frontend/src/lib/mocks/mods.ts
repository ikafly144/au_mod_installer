// Mod情報の型定義
export interface ModInfo {
    id: string;
    name: string;
    description: string;
    version: string;
    author: string;
    installed: boolean;
    updateAvailable: boolean;
    imageUrl?: string;
}

// モックデータ生成関数
export function getMods(): ModInfo[] {
    return Array.from({ length: 10 }, (_, i) => ({
        id: `mod-${i + 1}`,
        name: `Sample Mod ${i + 1}`,
        description: `これは Sample Mod ${i + 1} の簡単な説明です。機能や特徴がここに記載されます。`,
        version: `1.${i}.0`,
        author: `Modder${i + 1}`,
        installed: i % 3 === 0, // 3つに1つはインストール済み
        updateAvailable: i % 4 === 0, // 4つに1つはアップデートあり
    }));
}
