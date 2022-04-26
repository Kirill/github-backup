// Copyright 2022 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Github-backup application save your github repository to local disk
//
// App use 'git' and 'gh' (github-cli) applications which shoud be preinstalled
// on the host. The 'git' should be configured to has access to your
// repositories by ssh. The 'gh' should be logged in to your github account
// before call this app.
//
// Application parameters:
//
//   -users  <[user-or-organisation-comma-separated-list]>
//   -limit  [user-repo-comma-separated-list]
//   -output [local-folder-name], default: ./repos
//
// Usage examples:
//
//   go run . -users=kirill-scherba -limit=kirill-scherba/teonet-go -output=./tmp
//
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

func main() {

	// Parse parameters
	var userslist, limitslist, output, maxrepo string
	var stars, starsonly bool
	flag.StringVar(&userslist, "users", "", "user or organisation comma separated list")
	flag.StringVar(&limitslist, "limit", "", "user/repository comma separated list to backup, all if empty")
	flag.StringVar(&output, "output", "repos", "local folder name to save repositories")
	flag.BoolVar(&stars, "stars", false, "backup starred repositories also")
	flag.BoolVar(&starsonly, "starsonly", false, "backup starred repositories only")
	flag.StringVar(&maxrepo, "maxrepo", "1000", "maximum number of users repositories to be cloned")
	flag.Parse()

	// Parse users and limit
	users := strings.Split(userslist, ",")
	var limit []string
	if len(strings.TrimSpace(limitslist)) != 0 {
		limit = strings.Split(limitslist, ",")
	}

	// Get list of repos with gh cli application
	for _, user := range users {
		if !starsonly {
			getRepos(output, strings.TrimSpace(user), maxrepo, limit)
		}
		if stars || starsonly {
			getStars(output, strings.TrimSpace(user), maxrepo, limit)
		}
	}
}

// Number of repositories to show in print
var reponum int

// getRepos get list of reopsitories and clone it
func getRepos(dir, user, maxrepo string, limit []string) (repos []string) {

	// Get list of reopsitories with gh
	out, err := exec.Command("gh", "repo", "list", user, "-L", maxrepo).Output()
	if err != nil {
		log.Fatal(err)
	}

	// Parse gh ouput
	strs := strings.Split(string(out), "\n")
	for i := range strs {
		// Skip empty string
		if len(strs[i]) == 0 {
			continue // or break because the last line of 'out' is empty
		}

		// Get first column from 'gh repo list' output, it's repo name
		words := strings.Split(strs[i], "\t")
		repos = append(repos, words[0])
	}

	// Clone repos
	cloneRepos(repos, limit, dir)

	return
}

// getStars get list of starred reopsitories and clone it
func getStars(dir, user, maxrepo string, limit []string) (repos []string) {

	// Get stars by github api
	resp, err := http.Get(fmt.Sprintf("https://api.github.com/users/%s/starred?page=1&per_page=%s", user, maxrepo))
	if err != nil {
		if err != nil {
			log.Printf("Can't get starred repos of %s: %s", user, err)
			return nil
		}
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Can't read response body: %s", err)
		return nil
	}

	// Umarshal github api output
	type starsData struct {
		FullName string `json:"full_name,omitempty"`
	}
	var jsonData []starsData
	if err := json.Unmarshal(body, &jsonData); err != nil {
		log.Printf("Can't parse response body to json: %s\n%s", err, string(body))
		return nil
	}

	// Parse github api output
	for i := range jsonData {
		repos = append(repos, jsonData[i].FullName)
	}

	// Clone repos
	cloneRepos(repos, limit, dir)

	return
}

// cloneRepos from list of full repo name
func cloneRepos(repos []string, limit []string, dir string) (cloned []string) {
	for _, repo := range repos {
		// All if 'limit' slice empty or if 'repo' exists in 'limit' slice
		if !(len(limit) == 0 || inSlise(repo, limit)) {
			continue
		}

		// Print repo name
		reponum++
		fmt.Printf("repo %3d: %s\n", reponum, repo)
		repos = append(repos, repo)

		// Clone repo
		_, err := exec.Command("git", "clone", "--mirror", "git@github.com:"+repo+".git", dir+"/"+repo+".git").Output()
		if err != nil {
			log.Fatal(err)
		}

		// Clone wiki repo
		err = exec.Command("git", "clone", "--mirror", "git@github.com:"+repo+".wiki.git", dir+"/"+repo+".wiki.git").Run()
		if err != nil {
			// log.Println(err)
		}
	}
	return
}

// inSlise return true if string 'el' exists in 'ar' string slice
func inSlise(el string, ar []string) bool {
	for i := range ar {
		if strings.TrimSpace(ar[i]) == el {
			return true
		}
	}
	return false
}
