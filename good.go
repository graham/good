package main

import (
	"fmt"
	"github.com/libgit2/git2go"
	"os"
	"path/filepath"
	"strings"
)

const (
	PATH_DATA = "~/Dropbox/gistory/"
)

func walk_branch(repo *git.Repository, branch *git.Branch) {
	walk, _ := repo.Walk()

	walk.Sorting(git.SortTopological)
	walk.Reset()
	walk.PushHead()
	walk.Iterate(func(commit *git.Commit) bool {
		fmt.Println(commit.Message())
		return true
	})
}

func process_repo(finalize chan int, repo_chan chan string) {
	for path := range repo_chan {
		repo, err := git.OpenRepository(path)
		if err != nil {
			fmt.Println("Counldn't find repo at ", path)
		} else {
			iter, _ := repo.NewBranchIterator(git.BranchLocal)
			b, _, berr := iter.Next()

			for berr == nil {
				walk_branch(repo, b)
				b, _, berr = iter.Next()
			}
		}
	}

	finalize <- 0
	close(finalize)
}

func find_repos(repo_chan chan string, path string) {
	abspath := os.Getenv("HOME") + "/" + path

	visit := func(hitpath string, f os.FileInfo, err error) error {
		sp := strings.Split(hitpath, ".")
		if sp[len(sp)-1] == "git" {
			repo_chan <- hitpath
		}
		return nil
	}

	filepath.Walk(abspath, visit)
}

func main() {
	PATH_SEARCH := []string{"Dropbox/golang/src/github.com/graham/good/"}
	finalize := make(chan int)
	repo_chan := make(chan string)

	go process_repo(finalize, repo_chan)

	for path_index := range PATH_SEARCH {
		find_repos(repo_chan, PATH_SEARCH[path_index])
	}
	close(repo_chan)

	<-finalize
}
