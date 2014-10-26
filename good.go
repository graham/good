package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/libgit2/git2go"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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

func walk_branch(repo *git.Repository, branch *git.Branch, f *bufio.Writer, search_email string) {
	walk, _ := repo.Walk()

	walk.Sorting(git.SortTopological)
	walk.PushRef(branch.Reference.Name())

	walk.Iterate(func(commit *git.Commit) bool {
		author := commit.Author()
		if len(search_email) > 0 && author.Email != search_email {
			return true
		}

		tree, _ := commit.Tree()
		//fileList := walk_tree(repo, tree, "")

		opts := git.DiffOptions{}
		changeTypes := make(map[string]int, 0)

		if commit.ParentCount() > 0 {
			for i := uint(0); i < commit.ParentCount(); i++ {
				parent := commit.Parent(i)
				oldTree, _ := parent.Tree()
				diff, _ := repo.DiffTreeToTree(oldTree, tree, &opts)

				_ = diff.ForEach(func(file git.DiffDelta, progress float64) (git.DiffForEachHunkCallback, error) {
					fullfilename := file.OldFile.Path
					sp := strings.Split(fullfilename, "/")
					filename := sp[len(sp)-1]
					sp2 := strings.Split(filename, ".")
					extension := sp2[len(sp2)-1]

					return func(hunk git.DiffHunk) (git.DiffForEachLineCallback, error) {
						return func(line git.DiffLine) error {
							if line.OldLineno == -1 {
								changeTypes["+"+extension] += 1
							}
							if line.NewLineno == -1 {
								changeTypes["-"+extension] += 1
							}
							return nil
						}, nil
					}, nil
				}, git.DiffDetailLines)
			}
		}

		j, _ := json.Marshal(changeTypes)

		row := fmt.Sprintf("%s,%s,%s,%d,%s,%s,%s\n",
			commit.Id(),
			repo.Path(),
			branch.Reference.Name(),
			author.When.Unix(),
			author.When.Local(),
			author.Email,
			j)

		f.WriteString(row)
		return true
	})
}

func process_repo(finalize chan int, repo_chan chan string, search_email string, save_file_path string) {
	f, ferr := os.Create(save_file_path)

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
				walk_branch(repo, b, writer, search_email)
				b, _, berr = iter.Next()
			}
		}
	}

	writer.Flush()

	finalize <- 0
	close(finalize)
}

func find_repos(repo_chan chan string, path string) {
	abspath := path

	visit := func(hitpath string, f os.FileInfo, err error) error {
		sp := strings.Split(hitpath, ".")
		if sp[len(sp)-1] == "git" {
			repo_chan <- hitpath
		}
		return nil
	}

	filepath.Walk(abspath, visit)
}

func analyize(filename string, in_last int64) {
	var lower_bound int64 = 0

	if in_last == -1 {
		// pass
	} else {
		lower_bound = time.Now().Unix() - int64(in_last*60*60*24)
	}

	file, _ := os.Open(filename)
	scanner := bufio.NewReader(file)

	changeTypes := make(map[string]int, 0)

	for line, _, err := scanner.ReadLine(); err == nil; line, _, err = scanner.ReadLine() {
		sp := strings.SplitN(string(line), ",", 7)
		localChange := make(map[string]int)

		commit_time, _ := strconv.ParseInt(sp[3], 10, 64)
		if commit_time >= lower_bound {
			json.Unmarshal([]byte(sp[6]), &localChange)

			for key, value := range localChange {
				changeTypes[key] += value
			}
		}

	}

	for key, value := range changeTypes {
		if key[0] == '+' {
			fmt.Printf("%14s | %d\n", key, value)
		}
	}
}

func main() {
	var fpath *string = flag.String("path", "./", "Path to search (default ./)")
	var femail *string = flag.String("email", "", "The author (by email) to search for.")
	var skip_search *int = flag.Int("skip", 0, "If 1 skip the scan step.")
	var lower_bound *int64 = flag.Int64("days", -1, "History in days to search.")

	flag.Parse()

	if len(*femail) == 0 {
		fmt.Println("usage: good --email=youremail@domain.com")
		fmt.Println("       good --path=path/to/dir/ --email=youremail@domain.com")
		fmt.Println(" additional args:")
		fmt.Println("            --days=<int> # in the last int days.")
		fmt.Println("            --skip=<int> # 1 to skip the scanning step.")
		return
	}

	path, _ := filepath.Abs(string(*fpath))
	email := strings.Replace(string(*femail), "\\@", "@", -1)

	save_file_path := os.Getenv("HOME") + "/commit_history_" + email + ".csv"

	if *skip_search != 1 {
		finalize := make(chan int)
		repo_chan := make(chan string)

		go process_repo(finalize, repo_chan, email, save_file_path)

		find_repos(repo_chan, path)
		close(repo_chan)
		<-finalize
	}

	analyize(save_file_path, *lower_bound)
}
