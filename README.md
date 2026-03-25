# tarm

Terraform モジュールの依存関係を解析し、変更されたファイルから影響を受ける root module を特定するツールです。

## クイックスタート

### GitHub Actions

```yaml
- uses: kzmshx/tarm@main
  with:
    root: ./infrastructure
    root-module-patterns: |
      infrastructure/environments/*/*
      infrastructure/stacks/*/*
    exclude-module-patterns: |
      infrastructure/modules/*
```

## GitHub Actions での使用方法

### 入力パラメータ

| パラメータ | 必須 | デフォルト | 説明 |
|-----------|------|-----------|------|
| `root` | No | `.` | 検索するルートディレクトリ |
| `root-module-patterns` | Yes | - | root module の glob パターン（改行区切り） |
| `exclude-module-patterns` | No | - | root module から除外する non-root module の glob パターン（改行区切り） |
| `changed-files` | No | - | 解析対象パス（改行区切り。未指定時は git diff で自動検出） |
| `detect-changes` | No | `true` | git diff による変更ファイルの自動検出を有効にするか |
| `base-ref` | No | `github.base_ref` | 変更検出のベース ref |
| `head-ref` | No | `github.head_ref` | 変更検出のヘッド ref |
| `output-format` | No | `github` | 出力形式（`github` または `json`） |
| `comment-pr` | No | `true` | PR に結果をコメント |

### 出力

| 出力 | 説明 |
|-----|------|
| `affected-modules` | 影響を受けるモジュールのスペース区切りリスト |
| `affected-modules-json` | 影響を受けるモジュールの詳細を含む JSON 配列 |
| `affected-count` | 影響を受けるモジュール数 |
| `has-affected-modules` | 影響を受けるモジュールが存在するかどうか（`true`/`false`） |
| `matrix` | GitHub Actions matrix 戦略用 JSON |
| `markdown-summary` | 影響を受けるモジュールのマークダウンサマリー |

### 完全な例

```yaml
name: Terraform Plan
on: pull_request

jobs:
  analyze:
    runs-on: ubuntu-latest
    outputs:
      matrix: ${{ steps.tarm.outputs.matrix }}
      has-affected-modules: ${{ steps.tarm.outputs.has-affected-modules }}
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: kzmshx/tarm@main
        id: tarm
        with:
          root: ./infrastructure
          root-module-patterns: |
            infrastructure/environments/*/*
            infrastructure/stacks/*/*
          exclude-module-patterns: |
            infrastructure/modules/*
          comment-pr: true

  plan:
    needs: analyze
    if: needs.analyze.outputs.has-affected-modules == 'true'
    strategy:
      matrix: ${{ fromJson(needs.analyze.outputs.matrix) }}
    runs-on: ubuntu-latest
    steps:
      - run: terraform plan ${{ matrix.module }}
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

`modules/database` が変更された場合、以下のような結果が出力されます：

```json
[
  {
    "path": "environments/dev/api",
    "affected_by": ["modules/database"]
  },
  {
    "path": "environments/prod/api",
    "affected_by": ["modules/database"]
  }
]
```

## 動作原理

1. 指定されたパターンに基づいて root module と non-root module を識別
2. Git diff を使用して変更されたファイルを検出
3. ローカル Terraform モジュール依存関係を解析（外部モジュールは無視）
4. モジュール間の依存グラフを構築し、循環依存をチェック
5. 変更されたファイルから影響を受ける root module を特定
6. 結果を JSON 形式とマークダウン形式で出力

**注意:** 非 .tf ファイル（Lambda ソースなど）は .tf ファイルを含む親ディレクトリまで遡って処理されます

## 機能

- ✅ ローカルモジュール依存関係解析
- ✅ 非 .tf ファイル処理（親ディレクトリへのエスカレーション）
- ✅ 循環依存関係の検出
- ✅ GitHub Actions 統合
- ✅ JSON およびマークダウン出力形式
- ✅ PR への自動コメント機能
- ❌ 外部モジュール（Registry、Git、S3）- 意図的に無視
