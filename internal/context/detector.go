package context

import "github.com/danjdewhurst/jot-cli/internal/model"

func AutoTags() []model.Tag {
	var tags []model.Tag

	if folder, err := DetectFolder(); err == nil && folder != "" {
		tags = append(tags, model.Tag{Key: "folder", Value: folder})
	}
	if repo, err := DetectRepo(); err == nil && repo != "" {
		tags = append(tags, model.Tag{Key: "git_repo", Value: repo})
	}
	if branch, err := DetectBranch(); err == nil && branch != "" {
		tags = append(tags, model.Tag{Key: "git_branch", Value: branch})
	}

	return tags
}
