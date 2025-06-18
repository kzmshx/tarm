# tarm

Terraform モジュールの依存関係を解析し、変更されたファイルから影響を受ける root module を特定するツールです。

## クイックスタート

### GitHub Actions

```yaml
- uses: kzmshx/tarm@main
  with:
    root: ./infrastructure
    entrypoints: infrastructure/environments/*/*
```

## GitHub Actions での使用方法

### 入力パラメータ

| パラメータ | 必須 | デフォルト | 説明 |
|-----------|------|-----------|------|
| `root` | No | `.` | 検索するルートディレクトリ |
| `entrypoints` | Yes | - | root module の glob パターン |
| `paths` | No | - | 解析対象パス（未指定時は変更ファイル自動検出） |
| `exclude` | No | `.terraform/**` | 除外パターン |
| `comment-pr` | No | `true` | PR に結果をコメント |

### 出力

| 出力 | 説明 |
|-----|------|
| `affected-modules` | 影響を受けるモジュールのスペース区切りリスト |
| `affected-count` | 影響を受けるモジュール数 |
| `matrix` | GitHub Actions matrix 戦略用 JSON |

### 完全な例

```yaml
name: Terraform Plan
on: pull_request

jobs:
  analyze:
    runs-on: ubuntu-latest
    outputs:
      matrix: ${{ steps.tarm.outputs.matrix }}
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: kzmshx/tarm@main
        id: tarm
        with:
          root: ./infrastructure
          entrypoints: |
            infrastructure/environments/*/*
            infrastructure/stacks/*/*

  plan:
    needs: analyze
    if: needs.analyze.outputs.matrix != '[]'
    strategy:
      matrix: ${{ fromJson(needs.analyze.outputs.matrix) }}
    runs-on: ubuntu-latest
    steps:
      - run: terraform plan ${{ matrix.path }}
```

## 使用例

### モノレポ構造

```text
infrastructure/
├── environments/
│   ├── dev/
│   │   ├── api/main.tf      # modules/database に依存
│   │   └── web/main.tf      # modules/auth に依存
│   └── prod/
│       ├── api/main.tf
│       └── web/main.tf
└── modules/
    ├── auth/main.tf
    ├── database/main.tf
    └── network/main.tf
```

### データベースモジュール変更時

```bash
tarm \
  --root ./infrastructure \
  --entrypoints "environments/*/*" \
  --paths modules/database
```

出力:

```text
environments/dev/api
environments/prod/api
```

## 動作原理

1. ローカル Terraform モジュール依存関係を解析（外部モジュールは無視）
2. モジュール間の依存グラフを構築
3. 変更されたファイルから影響を受ける root module を特定
4. CI/CD パイプライン最適化のため結果を出力

**注意:** 非 .tf ファイル（Lambda ソースなど）は .tf ファイルを含む親ディレクトリまで遡って処理されます

## 機能

- ✅ ローカルモジュール依存関係解析
- ✅ 非 .tf ファイル処理（親ディレクトリへのエスカレーション）
- ✅ GitHub Actions 統合
- ✅ JSON およびテキスト出力形式
- ❌ 外部モジュール（Registry、Git、S3）- 意図的に無視
