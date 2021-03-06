package sdp

import (
	"github.com/concourse-friends/concourse-builder/library"
	"github.com/concourse-friends/concourse-builder/library/image"
	"github.com/concourse-friends/concourse-builder/library/primitive"
	"github.com/concourse-friends/concourse-builder/model"
	"github.com/concourse-friends/concourse-builder/project"
	"github.com/concourse-friends/concourse-builder/resource"
	"github.com/jinzhu/copier"
)

type BranchesJobArgs struct {
	ConcourseBuilderGit     *project.Resource
	ImageRegistry           *image.Registry
	GoImage                 *project.Resource
	ResourceRegistry        *project.ResourceRegistry
	Concourse               *primitive.Concourse
	TargetGitRepo           *primitive.GitRepo
	Environment             map[string]interface{}
	GenerateProjectLocation project.IRun
}

func taskObtainBranches(args *BranchesJobArgs, branchesDir *project.TaskOutput) *project.TaskStep {
	gitImageJobArgs := &library.GitImageJobArgs{}
	copier.Copy(gitImageJobArgs, args)

	gitImage := library.GitImageJob(gitImageJobArgs)

	gitImageResource := args.ResourceRegistry.JobResource(gitImage, true, nil)

	targetGitResource := &project.Resource{
		Name: "target-git",
		Type: resource.GitMultibranchResourceType.Name,
		Source: &library.GitMultiSource{
			Repo:     args.TargetGitRepo,
			Branches: "master|release[/].*|feature[/].*|task[/].*",
		},
	}

	targetGitJobResource := args.ResourceRegistry.JobResource(targetGitResource, true, nil)

	environment := map[string]interface{}{
		"GIT_REPO_DIR": &primitive.Location{
			Volume: targetGitJobResource,
		},
		"GIT_PRIVATE_KEY": args.TargetGitRepo.PrivateKey,
		"OUTPUT_DIR":      branchesDir.Path(),
	}

	task := &project.TaskStep{
		Platform: model.LinuxPlatform,
		Name:     "obtain branches",
		Image:    gitImageResource,
		Run: &primitive.Location{
			Volume: &primitive.Directory{
				Root: "/bin/git",
			},
			RelativePath: "obtain_branches.sh",
		},

		Environment: environment,
		Outputs: []project.IOutput{
			branchesDir,
		},
	}

	return task
}

func taskPreparePipelines(args *BranchesJobArgs, branchesDir *project.TaskOutput, pipelinesDir *project.TaskOutput) *project.TaskStep {
	goImageResource := args.ResourceRegistry.JobResource(args.GoImage, true, nil)

	environment := make(map[string]interface{})
	for k, v := range args.Environment {
		environment[k] = v
	}

	environment[BranchesFileEnvVar] = &primitive.Location{
		Volume:       branchesDir,
		RelativePath: "branches",
	}

	task := &project.TaskStep{
		Platform:    model.LinuxPlatform,
		Name:        "prepare pipelines",
		Image:       goImageResource,
		Run:         args.GenerateProjectLocation,
		Environment: environment,
		Outputs: []project.IOutput{
			pipelinesDir,
		},
	}

	return task
}

func taskCreateMissingPipelines(args *BranchesJobArgs, pipelinesDir *project.TaskOutput) *project.TaskStep {
	flyImageJobArgs := &library.FlyImageJobArgs{}
	copier.Copy(flyImageJobArgs, args)

	flyImage := library.FlyImageJob(flyImageJobArgs)

	flyImageResource := args.ResourceRegistry.JobResource(flyImage, true, nil)

	task := &project.TaskStep{
		Platform: model.LinuxPlatform,
		Name:     "create missing pipelines",
		Image:    flyImageResource,
		Run: &primitive.Location{
			Volume: &primitive.Directory{
				Root: "/bin/fly",
			},
			RelativePath: "create_missing_pipelines.sh",
		},
		Environment: map[string]interface{}{
			"PIPELINES": &primitive.Location{
				Volume: pipelinesDir,
			},
		},
	}

	args.Concourse.Environment(task.Environment)

	return task
}

func taskRemoveNotNeededPipelines(args *BranchesJobArgs, pipelinesDir *project.TaskOutput, branchesDir *project.TaskOutput) *project.TaskStep {
	flyImageJobArgs := &library.FlyImageJobArgs{}
	copier.Copy(flyImageJobArgs, args)

	flyImage := library.FlyImageJob(flyImageJobArgs)

	flyImageResource := args.ResourceRegistry.JobResource(flyImage, true, nil)

	task := &project.TaskStep{
		Platform: model.LinuxPlatform,
		Name:     "remove not needed pipelines",
		Image:    flyImageResource,
		Run: &primitive.Location{
			Volume: &primitive.Directory{
				Root: "/bin/fly",
			},
			RelativePath: "remove_not_needed_pipelines.sh",
		},
		Environment: map[string]interface{}{
			"PIPELINES": &primitive.Location{
				Volume: pipelinesDir,
			},
			"BRANCHES_DIR":   branchesDir.Path(),
			"PIPELINE_REGEX": ".*-sdpb$",
		},
	}

	args.Concourse.Environment(task.Environment)

	return task
}

func BranchesJob(args *BranchesJobArgs) *project.Job {
	branchesDir := &project.TaskOutput{
		Directory: "branches",
	}

	taskObtainBranches := taskObtainBranches(args, branchesDir)

	pipelinesDir := &project.TaskOutput{
		Directory: "pipelines",
	}

	taskPreparePipelines := taskPreparePipelines(args, branchesDir, pipelinesDir)

	taskCreateMissingPipelines := taskCreateMissingPipelines(args, pipelinesDir)

	taskRemoveNotNeededPipelines := taskRemoveNotNeededPipelines(args, pipelinesDir, branchesDir)

	branchesJob := &project.Job{
		Name:   project.JobName("branches"),
		Groups: project.JobGroups{},
		Steps: project.ISteps{
			taskObtainBranches,
			taskPreparePipelines,
			taskCreateMissingPipelines,
			taskRemoveNotNeededPipelines,
		},
	}

	return branchesJob
}
