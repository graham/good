package main

import (
	"bufio"
	"fmt"
	"github.com/libgit2/git2go"
	"os"
	"path/filepath"
	"strings"
)

const (
	PATH_DATA = "Dropbox/gistory/"
)

func walk_tree(repo *git.Repository, tree *git.Tree, prefix string) string {
	fileList := []string{}

	tree.Walk(func(s string, t *git.TreeEntry) int {

		if t.Type == git.ObjectBlob {
			filename := prefix + "/" + t.Name
			fileList = append(fileList, filename)
		} else if t.Type == git.ObjectTree {
			newTree, _ := repo.LookupTree(t.Id)
			fileList = append(fileList, walk_tree(repo, newTree, t.Name))
		}
		return 1
	})

	return strings.Join(fileList, "|")
}

func walk_branch(repo *git.Repository, branch *git.Branch, f *bufio.Writer) {
	walk, _ := repo.Walk()

	walk.Sorting(git.SortTopological)
	walk.PushRef(branch.Reference.Name())

	walk.Iterate(func(commit *git.Commit) bool {
		author := commit.Author()
		if author.Email != "graham.abbott@gmail.com" {
			return true
		}

		tree, _ := commit.Tree()
		fileList := walk_tree(repo, tree, "")

		opts := git.DiffOptions{}

		//changeTypes := make(map[string]int)

		if commit.ParentCount() > 0 {
			for i := uint(0); i < commit.ParentCount(); i++ {
				parent := commit.Parent(i)
				oldTree, _ := parent.Tree()
				diff, _ := repo.DiffTreeToTree(oldTree, tree, &opts)

				files := make([]string, 0)
				hunks := make([]git.DiffHunk, 0)
				_ = diff.ForEach(func(file git.DiffDelta, progress float64) (git.DiffForEachHunkCallback, error) {
					files = append(files, file.OldFile.Path)
					return func(hunk git.DiffHunk) (git.DiffForEachLineCallback, error) {
						hunks = append(hunks, hunk)
						return func(line git.DiffLine) error {
							return nil
						}, nil
					}, nil
				}, git.DiffDetailLines)

				fmt.Printf("%d/%d/%s", hunks[0].OldLines, hunks[0].NewLines, hunks[0].Header)
			}
		}

		row := fmt.Sprintf("%s,%s,%d,%s,%s,%s\n",
			repo.Path(),
			branch.Reference.Name(),
			author.When.Unix(),
			author.When.Local(),
			author.Email,
			fileList)

		f.WriteString(row)
		return true
	})
}

func process_repo(finalize chan int, repo_chan chan string) {
	f, ferr := os.Create(os.Getenv("HOME") + "/" + PATH_DATA + "commit_history.csv")

	if ferr != nil {
		fmt.Println(ferr)
		return
	}

	defer f.Close()
	writer := bufio.NewWriter(f)

	for path := range repo_chan {
		repo, err := git.OpenRepository(path)
		if err != nil {
			fmt.Println("Counldn't find repo at ", path)
		} else {
			iter, _ := repo.NewBranchIterator(git.BranchLocal)
			b, _, berr := iter.Next()

			for berr == nil {
				walk_branch(repo, b, writer)
				b, _, berr = iter.Next()
			}
		}
	}

	writer.Flush()

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
