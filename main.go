package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/HandyGold75/GOLib/scheduler"

	"github.com/HandyGold75/GOLib/logger"

	"github.com/alexflint/go-arg"
)

type (
	Repos struct {
		Repos []Repo `json:"repos"`
	}

	Repo struct {
		// Name of the Borg Repo.
		Name string `json:"name"`

		// Password of the Borg Repo.
		Psw string `json:"psw"`

		// Paths to add to the Borg Repo.
		Sources []string `json:"sources"`

		// Paths to exclude from the Borg Repo.
		Excludes []string `json:"excludes"`
	}
)

var (
	scedule = scheduler.Scedule{
		Months:  []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
		Weeks:   []int{1, 2, 3, 4, 5},
		Days:    []int{0, 1, 2, 3, 4, 5, 6},
		Hours:   []int{4},
		Minutes: []int{0},
	}

	args struct {
		RepoPath    string `arg:"-r,--repopath" default:"/disk1" help:"Specify path to repo directory."`
		BorgPath    string `arg:"-b,--borgpath" default:"/bin/borg" help:"Specify path to borg."`
		Compression string `arg:"-c,--compression" default:"zstd,22" help:"Select compression algorithm."`
		Awake       bool   `arg:"-a,--awake" help:"Prevent shutdown after backup has completed."`
		Test        bool   `arg:"-t,--test" help:"Prevents any changes to repos"`
	}

	Config = Repos{
		Repos: []Repo{{
			Name:     "",
			Psw:      "",
			Sources:  []string{},
			Excludes: []string{},
		}},
	}

	lgr = logger.New("borgbackup.log")
)

func loadConfig() error {
	execPath, err := os.Executable()
	if err != nil {
		return err
	}
	execPathSplit := strings.Split(strings.ReplaceAll(execPath, "\\", "/"), "/")
	execPath = strings.Join(execPathSplit[:len(execPathSplit)-1], "/")

	bytes, err := os.ReadFile(execPath + "/borgbackup.json")
	if err != nil {
		return err
	}
	if err := json.Unmarshal(bytes, &Config); err != nil {
		return err
	}
	return nil
}

func verifyVars() error {
	if dir, err := os.Stat(args.RepoPath); os.IsNotExist(err) || !dir.IsDir() {
		return errors.New("RepoPath \"" + args.RepoPath + "\" does not exists or is not a directory!")
	}

	if file, err := os.Stat(args.BorgPath); os.IsNotExist(err) || file.IsDir() {
		return errors.New("BorgPath \"" + args.BorgPath + "\" does not exists or is a directory!")
	}

	allowedCompAlgs := []string{"", "lz4"}
	for i := 1; i < 23; i++ {
		allowedCompAlgs = append(allowedCompAlgs, "zstd,"+strconv.Itoa(i))
	}
	for i := 0; i < 10; i++ {
		allowedCompAlgs = append(allowedCompAlgs, "zlib,"+strconv.Itoa(i))
	}
	for i := 0; i < 10; i++ {
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
				if err := exec.Command("bash", "-c", borgArgsWrapped).Run(); err != nil {
					lgr.Log("high", "Backup failed", repo.Name, err)
				} else {
					lgr.Log("high", "Backup success", repo.Name)
				}
			}
			ch <- name
		}(repo.Name, ch)
	}

	for i := 0; i < execCount; i++ {
		lgr.Log("low", "Done backup", <-ch)
	}
}

func main() {
	arg.MustParse(&args)
	if err := verifyVars(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	lgr.UseSeperators = false
	lgr.CharCountPerMsg = 20

	if err := loadConfig(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	for {
		nextBackup := time.Now()
		if err := scheduler.SetNextTime(&nextBackup, &scedule); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		if time.Until(nextBackup) > time.Duration(0) {
			lgr.Log("low", "Backup sceduled for", nextBackup.Format("2006-Jan-02 15:04:05"))
			scheduler.SleepFor("Next backup in: ", time.Until(nextBackup), time.Second*time.Duration(1))
		}

		runBackup()

		scheduler.SleepFor("Sleeping for: ", time.Until(nextBackup)+(time.Minute*time.Duration(1)), time.Second*time.Duration(1))

		if args.Awake {
			continue
		}

		lgr.Log("medium", "Execute", "shutdown")
		err := exec.Command("sudo", "shutdown").Run()
		if err != nil {
			lgr.Log("high", "Failed to shutdown", err)
			os.Exit(1)
		}
		os.Exit(0)
	}
}
