# MODREPO

[English](README.md) | [日本語](README_JA.md)

MODREPO は、PCゲーム **「R.E.P.O.」**（Semiwork Studios）向けのスタンドアロン MOD ランチャー／マネージャーです。
[Thunderstore](https://thunderstore.io/c/repo/) の MOD を直接検索・インストール・管理し、プロファイルごとに分離された環境で手軽にプレイできます。

## 主な特徴

- **ゲームディレクトリを汚さない設計 (Zero Pollution)**: Unity Mono Doorstop の仕組みと `windows.SetDllDirectory` および起動引数を活用し、プロファイルディレクトリから BepInEx を直接読み込みます。R.E.P.O. 本体のゲームインストール先ディレクトリには一切ファイルを書き込まず、クリーンな状態を保ちます。
- **Thunderstore API 統合**: 公式 Thunderstore REPO コミュニティ API (`https://thunderstore.io/c/repo/`) から MOD リスト、詳細情報、依存関係、ZIP パッケージを高速かつ ETag キャッシュ付きで自動取得します。
- **プロファイル管理**: MOD の組み合わせをプロファイルとして複数管理可能。`.repopack` アーカイブファイルとしてエクスポート・インポート・共有が可能です。
- **BepInEx 自動セットアップ**: プロファイル作成時に Unity Mono 向け BepInEx パック（`BepInEx-BepInExPack`）を自動的にセットアップします。

## インストール

### 最新リリース

MODREPO の最新バージョンは [リリースページ](https://github.com/ikafly144/modrepo/releases/latest) からダウンロードできます。
Windows 版リリースは MSI インストーラーとして配布されています。

### ソースからのビルド

ソースコードからビルドするには、[Go](https://golang.org/dl/)（1.27以上推奨）および C コンパイラ（CGo / Fyne GUI用）がインストールされている必要があります。

```bash
git clone https://github.com/ikafly144/modrepo.git
cd modrepo
go build ./client
```

テストの実行:

```bash
go test ./...
```

## プロファイルアーカイブ形式

プロファイルの共有・バックアップには `.repopack` ファイル形式を使用します（`modrepo.profile.json` とアイコンを含む標準 ZIP 形式）。

## ライセンス

MODREPO は GNU General Public License v3.0 のもとでライセンスされています。詳細については [LICENSE](LICENSE) ファイルを参照してください。
