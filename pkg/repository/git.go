// Copyright 2023 The Okteto Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package repository

import (
	"context"
	"fmt"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"os/exec"
)

type gitRepoController struct {
	path       string
	repoGetter repositoryGetterInterface
}

func newGitRepoController() gitRepoController {
	return gitRepoController{
		repoGetter: gitRepositoryGetter{},
	}
}

type CleanStatus struct {
	IsClean bool
	Err     error
}

// IsClean checks if the repository have changes over the commit
//func (r gitRepoController) isClean(ctx context.Context) (bool, error) {
//	ctxWithTimeout, cancel := context.WithTimeout(ctx, 1*time.Second)
//	defer cancel()
//
//	ch := make(chan CleanStatus)
//
//	benchmark.StartTimer("4.5_before_isClean_status")
//	repo, err := r.repoGetter.get(r.path)
//	if err != nil {
//		ch <- CleanStatus{
//			IsClean: false,
//			Err:     fmt.Errorf("failed to analyze git repo: %w", err),
//		}
//	}
//	worktree, err := repo.Worktree()
//	if err != nil {
//		ch <- CleanStatus{
//			IsClean: false,
//			Err:     fmt.Errorf("failed to infer the git repo's current branch: %w", err),
//		}
//	}
//	benchmark.StopTimer("4.5_before_isClean_status")
//
//	go func() {
//		benchmark.StartTimer("4.5_isClean_status")
//		status, err := worktree.Status()
//		if err != nil {
//			ch <- CleanStatus{
//				IsClean: false,
//				Err:     fmt.Errorf("failed to infer the git repo's status: %w", err),
//			}
//			return
//		}
//		benchmark.StopTimer("4.5_isClean_status")
//
//		ch <- CleanStatus{status.IsClean(), nil}
//	}()
//
//	select {
//	case <-ctxWithTimeout.Done():
//		fmt.Println("RUNNING OUT OF TIME!!!")
//		s, err := status(worktree)
//		if err != nil {
//			return false, fmt.Errorf("failed to infer the git repo's status: %w", err)
//		}
//		return s.IsClean(), ctxWithTimeout.Err()
//	case res := <-ch:
//		return res.IsClean, res.Err
//	}
//}

func (r gitRepoController) isClean(ctx context.Context) (bool, error) {
	repo, err := r.repoGetter.get(r.path)
	if err != nil {
		return false, fmt.Errorf("failed to analyze git repo: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		// TODO: fix msg
		return false, fmt.Errorf("failed to analyze git repo: %w", err)
	}

	status, err := getGitStatus(worktree)
	if err != nil {
		// TODO: fix msg
		return false, fmt.Errorf("failed to analyze git repo: %w", err)
	}

	fmt.Println(status)

	return status.IsClean(), nil
}

// Alternative method to determine file status. Modified from original
// version which was part of the following pull request.
// https://github.com/zricethezav/gitleaks/pull/463
func getGitStatus(wt gitWorktreeInterface) (gitStatusInterface, error) {
	c := exec.Command("git", "status", "--porcelain", "-z")
	c.Dir = wt.GetRoot()
	out, err := c.Output()
	if err != nil {
		return gitStatusInterface{status: string(out)}, nil
	}

	//if len(out) == 0 {
	//    return true, nil
	//}
	//lines := strings.Split(string(out), "\000")
	//status := make(map[string]*git.FileStatus, len(lines))
	//
	//for _, line := range lines {
	//	if len(line) == 0 {
	//		continue
	//	}
	//
	//	ltrim := strings.TrimLeft(line, " ")
	//
	//	pathStatusCode := strings.SplitN(ltrim, " ", 2)
	//	if len(pathStatusCode) != 2 {
	//		continue
	//	}
	//
	//	statusCode := []byte(pathStatusCode[0])[0]
	//	path := strings.Trim(pathStatusCode[1], " ")
	//
	//	status[path] = &git.FileStatus{
	//		Staging: git.StatusCode(statusCode),
	//	}
	//}

	return gitStatusInterface{status: string(out)}, nil
}

func (gs gitStatusInterface) IsClean() bool {
	if len(gs.status) == 0 {
		return true
	}

	// TODO: check lines

	return false
}

// GetSHA returns the last commit sha of the repository
func (r gitRepoController) getSHA() (string, error) {
	isClean, err := r.isClean(context.TODO())
	if err != nil {
		return "", fmt.Errorf("failed to check if repo is clean: %w", err)
	}
	if !isClean {
		return "", nil
	}
	repo, err := r.repoGetter.get(r.path)
	if err != nil {
		return "", fmt.Errorf("failed to analyze git repo: %w", err)
	}
	head, err := repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to analyze git repo: %w", err)
	}
	return head.Hash().String(), nil
}

type repositoryGetterInterface interface {
	get(path string) (gitRepositoryInterface, error)
}

type gitRepositoryGetter struct{}

func (gitRepositoryGetter) get(path string) (gitRepositoryInterface, error) {
	repo, err := git.PlainOpen(path)
	if err != nil {
		return nil, err
	}
	return oktetoGitRepository{repo: repo}, nil
}

type oktetoGitRepository struct {
	repo *git.Repository
}

func (ogr oktetoGitRepository) Worktree() (gitWorktreeInterface, error) {
	worktree, err := ogr.repo.Worktree()
	if err != nil {
		return nil, err
	}
	return oktetoGitWorktree{worktree: worktree}, nil
}

func (ogr oktetoGitRepository) Head() (*plumbing.Reference, error) {
	return ogr.repo.Head()
}

type oktetoGitWorktree struct {
	worktree *git.Worktree
}

func (ogr oktetoGitWorktree) Status() (gitStatusInterface, error) {
	//status, err := ogr.worktree.Status()
	//if err != nil {
	//	return gitStatusInterface{}, err
	//}
	return gitStatusInterface{status: ""}, nil
}

//func (ogr oktetoGitWorktree) Filesystem() fsStatusInterface {
//	return ogr.worktree.Filesystem
//}

func (ogr oktetoGitWorktree) GetRoot() string {
	return ogr.worktree.Filesystem.Root()
}

//func (fs fsStatus) Root() string {
//	return fs.Root()
//}

// TODO: remove?
type oktetoGitStatus struct {
	status git.Status
}

func (ogs oktetoGitStatus) IsClean() bool {
	return ogs.status.IsClean()
}

type gitRepositoryInterface interface {
	Worktree() (gitWorktreeInterface, error)
	Head() (*plumbing.Reference, error)
}
type gitWorktreeInterface interface {
	Status() (gitStatusInterface, error)
	GetRoot() string
}
type gitStatusInterface struct {
	status string
}

//
//type fsStatusInterface interface {
//	Root() string
//}

//type fsStatus struct {
//	text string
//}
