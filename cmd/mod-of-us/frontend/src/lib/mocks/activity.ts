// アクティビティの型定義
export interface Activity {
    id: string;
    type: 'detect' | 'install' | 'update' | 'error';
    message: string;
    timestamp: Date;
    read: boolean;
}

// モックデータ生成関数
export function getRecentActivities(): Activity[] {
    return [
        {
            id: '1',
            type: 'detect',
            message: 'AmongUs v2024.x が検出されました',
            timestamp: new Date(),
            read: false
        },
        {
            id: '2',
            type: 'install',
            message: 'TownOfHost をインストールしました',
            timestamp: new Date(Date.now() - 1000 * 60 * 60 * 2), // 2時間前
            read: true
        },
        {
            id: '3',
            type: 'update',
            message: 'ExtremeRoles のアップデートが利用可能です',
            timestamp: new Date(Date.now() - 1000 * 60 * 60 * 24), // 1日前
            read: false
        }
    ];
}
