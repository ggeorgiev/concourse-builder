package primitive

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitBranch_BaseBranch(t *testing.T) {
	branch := &GitBranch{
		Name: "branch#base",
	}

	assert.Equal(t, "base", branch.BaseBranch())

	branch = &GitBranch{
		Name: "branch",
	}

	assert.Equal(t, "master", branch.BaseBranch())
}

func TestGitBranch_FriendlyName(t *testing.T) {
	branch := &GitBranch{
		Name: "branch#base",
	}

	assert.Equal(t, "branch", branch.FriendlyName())

	branch = &GitBranch{
		Name: "branch",
	}

	assert.Equal(t, "branch", branch.FriendlyName())
}