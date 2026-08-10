# Mod of Us

[English](README.md) | [日本語](README_JA.md)

Mod of Us は、ゲーム「Among Us」のための Mod マネージャーです。プレイヤーが簡単に多様な Mod をインストール、管理、切り替えできるようにし、ゲーム体験を向上させます。

## インストール

### 最新リリース

Mod of Us の最新バージョンは [リリースページ](https://github.com/ikafly144/au_mod_installer/releases/latest) からダウンロードできます。
Windows 版リリースは MSI インストーラーとして配布されています。

### ソースからのビルド

Mod of Us をソースからビルドするには、[Go](https://golang.org/dl/) がインストールされていることを確認してください。その後、リポジトリをクローンして以下のコマンドを実行します:

```bash
git clone https://github.com/ikafly144/au_mod_installer.git
cd au_mod_installer
```

ビルド時には Discord SDK の DLL が必要です。プライベートリポジトリ `ikafly144/mus-libs` から `discord_partner_sdk.dll` を取得し、`lib/` に配置してください。

```bash
go build ./client
```

## 多言語対応

Mod of Us は複数言語をサポートしています。翻訳の追加や修正を行うには、`client/locales` ディレクトリ内のファイルを編集してください。言語ごとに固有の JSON ファイルがあり、翻訳用のキーと値のペアを追加できます。

## 貢献

貢献を歓迎します！Mod of Us に貢献したい場合は、リポジトリをフォークし、変更を含めたプルリクエストを作成してください。既存のコードスタイルに従い、新機能に対するテストを含めるようにしてください。

## ライセンス

Mod of Us は GNU General Public License v3.0 のもとでライセンスされています。詳細については [LICENSE](LICENSE) ファイルを参照してください。
