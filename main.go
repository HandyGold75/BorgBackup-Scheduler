package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/HandyGold75/GOLib/argp"
	"github.com/HandyGold75/GOLib/cfg"
	"github.com/HandyGold75/GOLib/logger"
	"github.com/HandyGold75/GOLib/scheduler"
)

type (
	Repo struct {
		Name     string
		Psw      string
		Sources  []string
		Excludes []string
	}
)

var (
	args = argp.ParseArgs(struct {
		Help        bool   `switch:"h,help"         opts:"help"         help:"Scheduler for BorgBackup"`
		RepoPath    string `switch:"r,-repopath"    default:"/disk1"    help:"Specify path to repo directory."`
		BorgPath    string `switch:"b,-borgpath"    default:"/bin/borg" help:"Specify path to borg."`
		Compression string `switch:"c,-compression" default:"zstd,22"   help:"Select compression algorithm."`
		Awake       bool   `switch:"a,-awake"                           help:"Prevent shutdown after backup has completed."`
		Verbose     bool   `switch:"v,-verbose"                         help:"Be verbose."`
		Test        bool   `switch:"t,-test"                            help:"Prevents any changes to repos"`
	}{})

	Config = struct {
		Repos    []Repo
		Schedule scheduler.Schedule
	}{
		Repos: []Repo{{
			Name:     "",
			Psw:      "",
			Sources:  []string{},
			Excludes: []string{},
		}},
		Schedule: scheduler.Schedule{
			Months:  []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
			Weeks:   []int{1, 2, 3, 4, 5},
			Days:    []int{0, 1, 2, 3, 4, 5, 6},
			Hours:   []int{4},
			Minutes: []int{0},
		},
	}

	lgr, _ = logger.NewRel("borgbackup.log")
)

func verifyVars() error {
	if dir, err := os.Stat(args.RepoPath); os.IsNotExist(err) || !dir.IsDir() {
		return errors.New("RepoPath \"" + args.RepoPath + "\" does not exists or is not a directory!")
	}

	if file, err := os.Stat(args.BorgPath); os.IsNotExist(err) || file.IsDir() {
		return errors.New("BorgPath \"" + args.BorgPath + "\" does not exists or is a directory!")
	}

	allowedCompAlgs := []string{"", "lz4"}
	for i := range 22 {
		allowedCompAlgs = append(allowedCompAlgs, "zstd,"+strconv.Itoa(i+1))
	}
	for i := range 10 {
		allowedCompAlgs = append(allowedCompAlgs, "zlib,"+strconv.Itoa(i))
	}
	for i := range 10 {
		allowedCompAlgs = append(allowedCompAlgs, "lzma,"+strconv.Itoa(i))
	}

	if !slices.Contains(allowedCompAlgs, args.Compression) {
		return errors.New("Compression \"" + args.Compression + "\" is not a valid algorithm!\nPlease choice from (eq. \"zstd,22\"): \"\" | lz4 | zstd,<1-22> | zlib,<0-9> | lzma,<0-9>")
	}

	return nil
}

func runBackup() {
	ch := make(chan string)
	execCount := 0

	for _, repo := range Config.Repos {
		if repo.Name == "" || repo.Psw == "" || len(repo.Sources) == 0 {
			lgr.Log("high", "Invalid config", repo.Name)
			continue
		}

		lgr.Log("medium", "Start backup", repo.Name)

		borgArgs := []string{"export", "BORG_PASSPHRASE=\"" + repo.Psw + "\"", "&&", args.BorgPath, "create", "--list", "-v", "-p", "-C", args.Compression, args.RepoPath + "/" + repo.Name + "::" + time.Now().Format("2006-Jan-02")}
		if args.Test {
			borgArgs = append(borgArgs, "--dry-run")
		}
		borgArgs = append(borgArgs, repo.Sources...)
		for _, exclude := range repo.Excludes {
			borgArgs = append(borgArgs, []string{"-e", exclude}...)
		}

		borgArgsWrapped := "echo '" + strings.Join(borgArgs, " ") + "' | sh"

		execCount++
		go func(name string, ch chan string) {
			lgr.Log("low", "Execute", "bash -c "+borgArgsWrapped)
			if !args.Test {
				cmd := exec.Command("bash", "-c", borgArgsWrapped)
				if args.Verbose {
					cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
				}
				if err := cmd.Run(); err != nil {
					lgr.Log("high", "Backup failed", repo.Name, err)
				} else {
					lgr.Log("high", "Backup success", repo.Name)
				}
			}
			ch <- name
		}(repo.Name, ch)
	}

	for range execCount {
		lgr.Log("low", "Done backup", <-ch)
	}
}

func main() {
	lgr.UseSeperators = false
	lgr.CharCountPerPart = 20

	if err := cfg.Load("borgbackup", &Config); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if err := verifyVars(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for {
		nextBackup := time.Now()
		if err := scheduler.SetNextTime(&nextBackup, Config.Schedule); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if time.Until(nextBackup) > time.Duration(0) {
			lgr.Log("low", "Backup sceduled for", nextBackup.Format("2006-Jan-02 15:04:05"))
			scheduler.SleepUntil("Next backup in: ", nextBackup, time.Second*time.Duration(1))
		}

		runBackup()

		scheduler.SleepFor("Sleeping for: ", time.Minute, time.Second)

		if args.Awake {
			continue
		}

		lgr.Log("medium", "Execute", "shutdown")
		if !args.Test {
			cmd := exec.Command("sudo", "shutdown")
			if args.Verbose {
				cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
			}
			err := cmd.Run()
			if err != nil {
				lgr.Log("high", "Failed to shutdown", err)
				os.Exit(1)
			}
		}
		os.Exit(0)
	}
}
