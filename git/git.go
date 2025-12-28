package git

import (
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

type TagInfo struct {
	Name   string
	Source string
	Target string
	TS     int64
}

func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func GetBranches() ([]string, error) {
	current, _ := GetCurrentBranch()
	cmd := exec.Command("git", "for-each-ref", "--sort=-committerdate", "--format=%(refname:short)", "refs/heads", "refs/remotes")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var branches []string
	for _, l := range lines {
		if l != "" && l != current {
			branches = append(branches, l)
		}
	}
	return branches, nil
}

func GetMergedTags() ([]TagInfo, error) {
	cmd := exec.Command("git", "tag", "-l", "merged*")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	re := regexp.MustCompile(`^merged\.?\{(.+)\}->\{(.+)\}@(\d+)$`)

	var detected []TagInfo
	for _, l := range lines {
		if l == "" {
			continue
		}
		matches := re.FindStringSubmatch(l)
		if len(matches) < 4 {
			continue
		}

		ts, _ := strconv.ParseInt(matches[3], 10, 64)
		detected = append(detected, TagInfo{
			Name:   l,
			Source: matches[1],
			Target: matches[2],
			TS:     ts,
		})
	}

	sort.Slice(detected, func(i, j int) bool {
		return detected[i].TS > detected[j].TS
	})

	return detected, nil
}

func TagBranch(selectedBranch string) (string, error) {
	currentBranch, err := GetCurrentBranch()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	timestamp := time.Now().Unix()
	tagName := fmt.Sprintf("merged.{%s}->{%s}@%d", selectedBranch, currentBranch, timestamp)

	cmd := exec.Command("git", "tag", tagName, selectedBranch)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git tag failed: %s: %s", err, string(output))
	}

	return tagName, nil
}

func CleanupTagsAndBranches(tags []string) error {
	re := regexp.MustCompile(`^merged\.?\{(.+)\}->\{.*\}@\d+$`)
	currentBranch, _ := GetCurrentBranch()

	for _, tag := range tags {
		matches := re.FindStringSubmatch(tag)
		var source string
		if len(matches) >= 2 {
			source = matches[1]
		}

		if source != "" && source == currentBranch {
			fmt.Printf("Warning: Skipped cleanup for tag '%s' because branch '%s' is currently checked out.\n", tag, source)
			continue
		}

		// Delete tag
		cmdTag := exec.Command("git", "tag", "-d", tag)
		if _, err := cmdTag.CombinedOutput(); err != nil {
			fmt.Printf("Warning: failed to delete tag %s\n", tag)
		} else {
			fmt.Printf("Deleted tag: %s\n", tag)
		}

		// Delete branch
		if source != "" {
			cmdBranch := exec.Command("git", "branch", "-d", source)
			if _, err := cmdBranch.CombinedOutput(); err != nil {
				fmt.Printf("Note: branch '%s' safe-delete failed. Skipping.\n", source)
			} else {
				fmt.Printf("Deleted branch: %s\n", source)
			}
		}
	}
	return nil
}
