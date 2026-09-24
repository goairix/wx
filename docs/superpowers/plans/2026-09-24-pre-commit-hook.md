# Pre-commit Hook Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 删除本地根目录示例程序，并建立版本化、自动启用且不会误暂存代码的 Go pre-commit 检查。

**Architecture:** `.githooks/pre-commit` 负责暂存区空白检查、Go 格式化、测试和 vet；`scripts/setup-hooks.sh` 只负责把当前仓库的 `core.hooksPath` 指向 `.githooks`。Hook 仅在 Go 相关文件变更时执行完整 Go 检查，并拒绝自动处理同时含有未暂存修改的 Go 文件。

**Tech Stack:** Bash、Git、Go 1.23 标准工具链。

---

### Task 1: 建立版本化 Hook

**Files:**
- Create: `.githooks/pre-commit`
- Create: `scripts/setup-hooks.sh`
- Delete: `pre-commit`

- [ ] **Step 1: 写入实现前验证并确认失败**

运行：

```bash
test -x .githooks/pre-commit && test -x scripts/setup-hooks.sh
```

预期：失败，因为两个脚本尚不存在。

- [ ] **Step 2: 创建 `.githooks/pre-commit`**

写入：

```bash
#!/usr/bin/env bash
set -euo pipefail

repository_root=$(git rev-parse --show-toplevel 2>/dev/null) || {
    printf 'pre-commit: not inside a Git repository\n' >&2
    exit 1
}
cd "$repository_root"

fail() {
    printf 'pre-commit: %s\n' "$1" >&2
    exit 1
}

git diff --cached --check || fail 'staged content has whitespace errors'

go_related=false
go_files=()
while IFS= read -r -d '' file; do
    case "$file" in
        *.go)
            go_related=true
            [[ -f "$file" ]] && go_files+=("$file")
            ;;
        go.mod | go.sum)
            go_related=true
            ;;
    esac
done < <(git diff --cached --name-only -z --diff-filter=ACMR)

if [[ "$go_related" != true ]]; then
    printf 'pre-commit: staged checks passed\n'
    exit 0
fi

command -v go >/dev/null 2>&1 || fail 'go is not installed or not in PATH'

if ((${#go_files[@]} > 0)); then
    command -v gofmt >/dev/null 2>&1 || fail 'gofmt is not installed or not in PATH'
    for file in "${go_files[@]}"; do
        git diff --quiet -- "$file" || fail "$file has unstaged changes; run gofmt and stage it manually"
    done
    gofmt -w "${go_files[@]}" || fail 'gofmt failed'
    git add -- "${go_files[@]}"
fi

git diff --cached --check || fail 'formatted staged content has whitespace errors'
printf 'pre-commit: running go test ./...\n'
go test ./... || fail 'go test ./... failed'
printf 'pre-commit: running go vet ./...\n'
go vet ./... || fail 'go vet ./... failed'
printf 'pre-commit: all checks passed\n'
```

- [ ] **Step 3: 创建 `scripts/setup-hooks.sh`**

写入：

```bash
#!/usr/bin/env bash
set -euo pipefail

repository_root=$(git rev-parse --show-toplevel 2>/dev/null) || {
    printf 'setup-hooks: not inside a Git repository\n' >&2
    exit 1
}
hook="$repository_root/.githooks/pre-commit"
if [[ ! -x "$hook" ]]; then
    printf 'setup-hooks: %s is missing or not executable\n' "$hook" >&2
    exit 1
fi

git -C "$repository_root" config core.hooksPath .githooks
printf 'setup-hooks: enabled .githooks for %s\n' "$repository_root"
```

- [ ] **Step 4: 删除旧脚本并验证语法**

运行：

```bash
bash -n .githooks/pre-commit
bash -n scripts/setup-hooks.sh
test ! -e pre-commit
```

预期：全部通过。

### Task 2: 删除根目录示例并更新说明

**Files:**
- Delete: `main.go`
- Modify: `.gitignore`
- Modify: `README.md`

- [ ] **Step 1: 删除本地 `main.go`**

删除根目录中被忽略的 `main.go`，并确认它未出现在 Git 历史或暂存区中。

- [ ] **Step 2: 移除忽略规则**

从 `.gitignore` 删除独立的 `main.go` 规则，不修改其它规则。

- [ ] **Step 3: 更新开发说明**

在 README 开发章节中加入新克隆初始化命令：

```bash
./scripts/setup-hooks.sh
```

说明初始化后每次提交会自动格式化已暂存 Go 文件并执行测试与 vet。

### Task 3: 启用并验证 Hook

**Files:**
- Modify local Git config: `.git/config`

- [ ] **Step 1: 为当前仓库启用 Hook**

运行：

```bash
./scripts/setup-hooks.sh
git config --local --get core.hooksPath
```

预期输出：`.githooks`。

- [ ] **Step 2: 在临时克隆验证轻量提交路径**

创建临时克隆，启用 Hook，暂存 Markdown 修改并执行 Hook。预期只进行暂存区检查并成功退出。

- [ ] **Step 3: 在临时克隆验证 Go 提交路径**

在临时克隆中暂存一个格式错误的 Go 文件并执行 Hook。预期文件被 `gofmt` 格式化、重新暂存，随后测试和 vet 通过。

- [ ] **Step 4: 验证部分暂存保护**

在临时克隆中暂存 Go 文件后继续产生未暂存修改，再执行 Hook。预期 Hook 拒绝提交，且未暂存修改没有进入索引。

- [ ] **Step 5: 运行发布检查**

运行：

```bash
go test -count=1 ./...
go vet ./...
git diff --check
```

预期：全部成功。
