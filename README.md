# modsniffer

**Helps you find smelly `go.mod` files based on a list of relevant packages you
would like to be up to date.**

```yaml
# example modsniffer config

preferredGoVersion: "1.20"
relevantPackages:
  - name: "k8s.io/api"
    preferredVersion: "0.27.0"
```

Based on the difference between the `preferredVersion` in the config file and
the actual version used in the checked `go.mod` file, `modsniffer` calculates an
overall score for the "outdatedness" of the project in question.

```yaml
# example output for a project, using the above config

  github.com/some/repo

  » Go Version 
    └─ actual:    1.19.0
    └─ preferred: 1.20.0
    └─ Score:     10
  » k8s.io/api
    └─ actual:    0.23.17
    └─ preferred: 0.27.0
    └─ Score:     40

  Final Score: 50

```

This can be helpful if you're maintaining a lot of Go projects with similar
dependencies and want to triage the worst of them.

## Usage

Put the above config in a file called `.modsniffer.yaml` and place it in your
`$HOME` directory (this is the default config location, you can set a custom one
using the `--config` flag). Add dependencies you want to check for. Then run
`modsniffer` like this:

```bash
go run main.go ./../path/to/project/
```
The file path that you pass as an argument should point to a local directory
containing a `go.mod` file.

`modsniffer` will eventually support checking remote repositories via `https`
or `ssh` as well, but this is not yet implemented (see To-Dos below).

For everything else, see the help:

```bash
go run main.go --help
```

## To-Dos

This tool is work in progress. Still to do:

* [ ] Remove `--verbose` flag and replace with proper output formatting (e.g.
  `json`, `yaml`, `tsv`, `pretty`)
* [ ] Add unit test cases
* [ ] Implement support for remote repositories (prefixed with `https://` and
  `ssh://`/`git@`)
* [ ] Get newest version of a checked package or Go version from upstream, e.g.
  [pkg.go.dev](https://pkg.go.dev/), only compare against a `desiredVersion` if
  one is specified
* [ ] How to handle logging properly in a CLI tool?
* [ ] Implement GitHub actions for automated tests, linting and versioned
  releases

## Ideas

* Recursive mode (sniff all `go.mod` files in all subdirectories). This would
  help if you need an overview of `~/dev/go-projects`.
* Batch mode (give `modsniffer` a list of repositories and get a ranked scoring
  result).
* Support mutliple scoring algorithms.
