# BorgBackup-Scheduler

Simple Borg Backup scheduler.

Use at your own risk! Tested for my use case only.
Do your own testing in a safe environment to ensure you don't lose your current data.

The schedule is configured in main.go (planned for move to JSON config).
Repos are configured in the config file.
Rest is configured by command line argument.

Config file template will be generated at first run in `~/.config/golib/borgbackup.json`.
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
  ],
  "Schedule": {
    "Months": [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12],
    "Weeks": [1, 2, 3, 4, 5],
    "Days": [0, 1, 2, 3, 4, 5, 6],
    "Hours": [4],
    "Minutes": [0]
  }
}
```

## Args

```text
Usage: borgbackup.bin [-h] [-r <string>] [-b <string>] [-c <string>] [-a] [-v] [-t]
        Scheduler for BorgBackup

Help
  -h -help          <bool>    (help)
RepoPath
  -r --repopath     <string>
        Specify path to repo directory.
BorgPath
  -b --borgpath     <string>
        Specify path to borg.
Compression
  -c --compression  <string>
        Select compression algorithm.
Awake
  -a --awake        <bool>
        Prevent shutdown after backup has completed.
Verbose
  -v --verbose      <bool>
        Be verbose.
Test
  -t --test         <bool>
        Prevents any changes to repos
```
