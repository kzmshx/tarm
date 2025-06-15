# Terraform Affected Root Modules (tarm) 最終仕様書

## 1. 概要

`tarm` は、Terraform モジュールの変更が及ぼす影響範囲を特定するツールです。指定されたファイルやディレクトリに依存している Terraform root module を抽出し、CI/CD パイプラインでの効率的な `terraform plan/apply` の実行を支援します。

### 1.1 基本コンセプト

- **Root Module**: `terraform init/plan/apply` を実行する対象ディレクトリ（Terraform 公式用語）
- **Module**: `module` ブロックで参照される再利用可能な Terraform 構成
- **依存グラフ**: モジュール間の参照関係を表す有向グラフ（DAG）

### 1.2 主要機能

- リポジトリ内の Terraform モジュール依存関係の解析
- 変更されたパスから影響を受ける root module の特定
- CLI および GitHub Actions での実行サポート

## 2. 技術仕様

### 2.1 解析対象

- **対象**: リポジトリ内のローカルモジュール参照のみ

  ```hcl
  module "example" {
    source = "./modules/example"     # ✅ 解析対象
    source = "../shared/module"      # ✅ 解析対象
  }
  ```

- **対象外**: 外部モジュール参照

  ```hcl
  module "vpc" {
    source = "terraform-aws-modules/vpc/aws"     # ❌ Registry
    source = "git::https://github.com/..."      # ❌ Git
    source = "s3::https://s3.amazonaws.com/..."  # ❌ S3
  }
  ```

### 2.2 非 `.tf` ファイルの扱い

サブディレクトリ内の非 `.tf` ファイル（Lambda コード、テンプレート等）は、上位の `.tf` を含むディレクトリまで昇格して影響判定：

```
modules/function/
├── main.tf              # モジュール本体
└── src/                 # .tf なし
    └── index.js         # 変更ファイル
```

→ `src/index.js` の変更は `modules/function` の変更として扱う

### 2.3 エラーハンドリング

- **循環参照**: 警告をログ出力し、処理は継続
- **シンボリックリンク**: リンクを辿る（実パスを記録し無限ループ防止）
- **不正な module source**: 警告を出して無視、処理は継続

### 2.4 制限事項

- サブディレクトリ外の依存は検知不可
- 動的な `source` パスの解決は行わない
- HCL の完全な評価（変数展開等）は実施しない

## 3. インターフェース仕様

### 3.1 CLI インターフェース

```bash
tarm \
  --root <path>                    # Terraform 探索の起点ディレクトリ
  --entrypoints <glob-pattern>...  # Root module の glob パターン（複数指定可）
  --paths <path>...                # 依存関係を調べる対象パス（複数指定可）
  --exclude <glob-pattern>...      # 除外パターン（! 演算子サポート）
  [--json]                         # JSON 形式で出力
```

#### 使用例

```bash
tarm \
  --root ./infra \
  --entrypoints "infra/environments/*/*" \
  --paths modules/network src/lambda/auth \
  --exclude ".terraform/** !infra/modules/critical/**"
```

#### 出力形式

**Markdown 形式（デフォルト）**:

```markdown
## infra/environments/dev/web
- modules/network

## infra/environments/stg/api
- modules/database
- modules/auth
```

**JSON 形式（--json）**:

```json
{
  "affected_modules": [
    {
      "path": "infra/environments/dev/web",
      "affected_by": ["modules/network"]
    },
    {
      "path": "infra/environments/stg/api",
      "affected_by": ["modules/database", "modules/auth"]
    }
  ]
}
```

### 3.2 GitHub Actions インターフェース

```yaml
- uses: your-org/tarm@v1
  with:
    root: ./infra
    entrypoints: |
      infra/environments/*/*
    paths: ${{ steps.changed-files.outputs.files }}
    exclude: .terraform/**
```

#### 入力パラメータ

| パラメータ      | 必須 | デフォルト                              | 説明                                     |
| --------------- | ---- | --------------------------------------- | ---------------------------------------- |
| `root`          | ✓    | `.`                                     | Terraform 探索の起点                     |
| `entrypoints`   | ✓    | -                                       | Root module の glob パターン             |
| `paths`         | -    | -                                       | 解析対象パス（指定なしの場合は自動検出） |
| `changed-files` | -    | `true`                                  | PR の変更ファイルを自動検出              |
| `exclude`       | -    | `.terraform/** **/.terragrunt-cache/**` | 除外パターン                             |
| `base-ref`      | -    | PR base SHA                             | 変更検出の基準                           |
| `output-format` | -    | `github`                                | 出力形式                                 |

#### 出力

| 名前                    | 説明                                    |
| ----------------------- | --------------------------------------- |
| `affected-modules`      | 影響 root module のスペース区切りリスト |
| `affected-modules-json` | JSON 配列形式の詳細情報                 |
| `affected-count`        | 影響を受ける root module 数             |
| `matrix`                | GitHub Actions matrix 戦略用 JSON       |
| `markdown-summary`      | PR コメント用 Markdown                  |

## 4. 実装技術

### 4.1 依存関係の解析

- **terraform-config-inspect** を使用して `.tf` ファイルを解析
- `module` ブロックの `source` 属性から依存関係を抽出
- ローカルパスのみを解決して依存グラフを構築

### 4.2 影響分析アルゴリズム

1. 全 `.tf` ファイルから module 依存グラフを構築（caller → callee）
2. 変更ファイルから上位の `.tf` 含有ディレクトリを特定
3. 依存グラフを逆方向に辿り、影響を受ける root module を列挙

### 4.3 Glob パターン処理

- 標準的な glob 構文（`*`, `**`, `?` 等）をサポート
- `!` プレフィックスで除外パターンからの復帰
- 例：`".terraform/** !infra/modules/important/**"`

### 4.4 ロギング

- エラーと警告のみ stderr に出力
- フォーマット例：

  ```
  WARN: Module source "./not-exist" not found in module "broken"
  WARN: Circular dependency detected: modules/a -> modules/b -> modules/a
  ERROR: Failed to parse /path/to/main.tf: ...
  ```

## 5. 動作環境

### 5.1 要件

- Go 1.19+
- Terraform 0.12+ (HCL2 syntax required)

### 5.2 Terraform バージョン互換性

`tarm` は [terraform-config-inspect](https://github.com/hashicorp/terraform-config-inspect) を使用しており、その対応バージョンに準拠します。

- サポート: Terraform 0.12.x - 1.x (HCL2)
- 非サポート: Terraform 0.11.x 以前 (HCL1)

## 6. 初期バージョンの制限

以下の機能は初期バージョンでは実装しません：

- 設定ファイル（`.tarmrc` 等）のサポート
- パフォーマンス最適化（並行処理、キャッシュ）
- 詳細なログレベル制御（--verbose, --quiet）
- 構造化ログ出力
- Terragrunt 対応

## 7. プロジェクト構成

Go 言語で実装し、以下の構成を採用：

- `cmd/tarm/`: CLI エントリーポイント
- `internal/`: 内部パッケージ群
  - `analyzer/`: 中核となる依存解析ロジック
  - `graph/`: 依存グラフのデータ構造と操作
  - `inspector/`: terraform-config-inspect ラッパー
  - `resolver/`: 各種パス解決ロジック
  - `cli/`: CLI 固有の処理
- `action/`: GitHub Actions 統合（TypeScript）
- `pkg/glob/`: 汎用 glob 処理ライブラリ
