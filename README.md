# BorgBackup-Scheduler

Simple Borg Backup scheduler.

Use at your own risk! Tested for my use case only.
Do your own testing in a safe environment to ensure you don't lose your current data.

The schedule is configured in main.go (planned for move to JSON config).
Repos are configured in the config file.
Rest is configured by command line argument.

Config file template will be generated at first run.
Configuration follows this pattern:

```json
{
  "repos": [
    {
      "name": "RepoName",
      "psw": "RepoPassword",
      "sources": ["/some/source/dir"],
      "excludes": ["/some/source/dir/*/.git"]
    }
  ]
}
```

Args:

```text
Usage: borgbackup.bin [--repopath REPOPATH] [--borgpath BORGPATH] [--compression COMPRESSION] [--awake] [--test]

Options:
  --repopath REPOPATH, -r REPOPATH
                         Specify path to repo directory. [default: /disk1]
  --borgpath BORGPATH, -b BORGPATH
                         Specify path to borg. [default: /bin/borg]
  --compression COMPRESSION, -c COMPRESSION
                         Select compression algorithm. [default: zstd,22]
  --awake, -a            Prevent shutdown after backup has completed.
  --test, -t             Prevents any changes to repos
  --help, -h             display this help and exit
```
