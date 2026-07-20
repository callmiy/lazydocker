# Personal fork workflow

This fork keeps upstream history easy to consume while providing a stable branch
for curated pull requests and personal changes.

## Branch contracts

- `master` mirrors `jesseduffield/lazydocker:master`. Do not add personal commits.
- `personal` is the supported, installed version of the fork.
- `import/pr-<number>-<slug>` evaluates an upstream pull request.
- `feature/<slug>` contains an original change.

The local `master` branch pulls from `upstream/master`; pushes still go to
`origin`. This makes an accidental `git pull` consume the correct history
without sending changes to the upstream repository.

## Remotes

```text
origin    callmiy/lazydocker
upstream  jesseduffield/lazydocker
```

Verify the configuration with:

```sh
git remote -v
git config --get remote.pushDefault
git branch -vv
```

## Import an upstream pull request

Treat every open pull request as untrusted code. Inspect its metadata and patch
before fetching or executing anything:

```sh
gh pr view <number> --repo jesseduffield/lazydocker
gh pr diff <number> --repo jesseduffield/lazydocker
```

Pay extra attention to changes under `.github/workflows`, scripts, Dockerfiles,
`go.mod`, `go.sum`, and `vendor`. Then start an isolated import:

```sh
git switch personal
scripts/personal/import-pr <number> <short-slug>
```

The helper requires a clean `personal` branch, fetches the pull-request head into
`upstream/pr/<number>`, creates the import branch, and prepares a no-commit merge.
It does not run fetched code or create the merge commit.

Review and validate the prepared merge:

```sh
git status
git diff --cached --stat
git diff --cached
GOFLAGS=-mod=vendor go test ./...
bash ./test.sh
GOFLAGS=-mod=vendor go build ./...
```

Adapt the change and add tests when necessary. Once satisfied:

```sh
git commit -m "Import upstream PR #<number>: <summary>"
git push -u origin "$(git branch --show-current)"
gh pr create --draft --base personal --fill
```

The merge commit retains the original commits and authors. If the change is not
worth keeping, abort it before leaving the branch:

```sh
git merge --abort
git switch personal
git branch -D import/pr-<number>-<short-slug>
```

## Add an original feature

```sh
git switch personal
git pull --ff-only origin personal
git switch -c feature/<slug>
```

Keep each feature independently reviewable. Run the same test and build commands
used for imports, then open a pull request targeting `personal`.

## Sync upstream

Update the pristine mirror first, then merge that known point into `personal`:

```sh
git fetch upstream --prune --tags
git switch master
git merge --ff-only upstream/master
git push origin master
git switch personal
git merge --no-ff master
GOFLAGS=-mod=vendor go test ./...
bash ./test.sh
git push origin personal
```

When an imported pull request is later merged upstream with the same commits, Git
normally recognizes the shared history. A squash or rebase merge can present the
same patch under new commit IDs; inspect and resolve those cases by resulting code
instead of blindly accepting either side.

## Releases and installation

Use fork-specific tags such as `v0.25.2-callmiy.1`, always created from
`personal`. Do not push a release tag yet: the inherited delivery workflow still
targets the upstream Homebrew tap and expects upstream-specific credentials. Add
a fork-owned GoReleaser configuration before publishing the first tag.

Until releases are configured, build the installed binary from `personal`:

```sh
git switch personal
GOFLAGS=-mod=vendor go build -o lazydocker .
```

Do not change the Go module path merely to install the binary. Retaining
`github.com/jesseduffield/lazydocker` keeps upstream imports and future merges
manageable; fork-owned release binaries provide the cleaner distribution path.
