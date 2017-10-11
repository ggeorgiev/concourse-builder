package sdp

import (
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/concourse-friends/concourse-builder/library"
	"github.com/concourse-friends/concourse-builder/library/image"
	"github.com/concourse-friends/concourse-builder/library/primitive"
	"github.com/concourse-friends/concourse-builder/project"
	"github.com/concourse-friends/concourse-builder/template/sdp_branch"
)

type Specification interface {
	Concourse() (*primitive.Concourse, error)
	DeployImageRegistry() (*image.Registry, error)
	LinuxImage(resourceRegistry *project.ResourceRegistry) (*project.Resource, error)
	GoImage(resourceRegistry *project.ResourceRegistry) (*project.Resource, error)
	ConcourseBuilderGit() (*project.Resource, error)
	GenerateProjectLocation(resourceRegistry *project.ResourceRegistry, branch *primitive.GitBranch) (project.IRun, error)
	TargetGitRepo() (*primitive.GitRepo, error)
	Environment() (map[string]interface{}, error)
}

const BranchesFileEnvVar = "BRANCHES_FILE"

func BootstrapBranches() ([]string, error) {
	branchesFile, exist := os.LookupEnv(BranchesFileEnvVar)
	if !exist {
		return nil, nil
	}
	branches, err := ioutil.ReadFile(branchesFile)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(branches), "\n"), err
}

func GenerateProject(specification Specification) (*project.Project, error) {
	prj := &project.Project{}

	branches, err := BootstrapBranches()
	if err != nil {
		return nil, err
	}

	for _, name := range branches {
		branch := &primitive.GitBranch{
			Name: name,
		}

		if !(branch.IsMaster() || branch.IsRelease() || branch.IsFeature() || branch.IsTask()) {
			log.Printf("Branch %s does not fit the expected name protocol", branch)
			continue
		}
		log.Printf("Preparing pipeline for branch %s", branch)

		branchSpecification := &BranchBootstrapSpecification{
			Specification: specification,
			TargetBranch:  branch,
		}

		project, err := sdpBranch.GenerateBootstrapProject(branchSpecification)
		if err != nil {
			return nil, err
		}

		prj.Pipelines = append(prj.Pipelines, project.Pipelines...)
	}

	concourseBuilderGit, err := specification.ConcourseBuilderGit()
	if err != nil {
		return nil, err
	}

	concourseBuilderBranch := concourseBuilderGit.Source.(*library.GitSource).Branch

	imageRegistry, err := specification.DeployImageRegistry()
	if err != nil {
		return nil, err
	}

	concourse, err := specification.Concourse()
	if err != nil {
		return nil, err
	}

	concourseBuilderPipeline := project.NewPipeline()
	concourseBuilderPipeline.AllJobsGroup = project.AllJobsGroupFirst
	concourseBuilderPipeline.Name = project.ConvertToPipelineName(concourseBuilderBranch.FriendlyName() + "-cb")

	concourseBuilderLinuxImage, err := specification.LinuxImage(concourseBuilderPipeline.ResourceRegistry)
	if err != nil {
		return nil, err
	}

	concourseBuilderPipeline.Jobs = project.Jobs{
		library.AllImages(&library.AllImagesArgs{
			LinuxImageResource:  concourseBuilderLinuxImage,
			ConcourseBuilderGit: concourseBuilderGit,
			ImageRegistry:       imageRegistry,
			ResourceRegistry:    concourseBuilderPipeline.ResourceRegistry,
			Concourse:           concourse,
		}),
	}

	mainPipeline := project.NewPipeline()
	mainPipeline.AllJobsGroup = project.AllJobsGroupFirst

	targetGit, err := specification.TargetGitRepo()
	if err != nil {
		return nil, err
	}

	mainPipeline.Name = project.ConvertToPipelineName(targetGit.FriendlyName() + "-sdp")

	if !concourseBuilderBranch.IsImage() {
		mainPipeline.ReuseFrom = append(mainPipeline.ReuseFrom, concourseBuilderPipeline.ResourceRegistry)
	}

	linuxImage, err := specification.LinuxImage(mainPipeline.ResourceRegistry)
	if err != nil {
		return nil, err
	}

	goImage, err := specification.GoImage(mainPipeline.ResourceRegistry)
	if err != nil {
		return nil, err
	}

	generateProjectLocation, err := specification.GenerateProjectLocation(mainPipeline.ResourceRegistry, nil)
	if err != nil {
		return nil, err
	}

	environment, err := specification.Environment()
	if err != nil {
		return nil, err
	}

	selfUpdateJob, _ := library.SelfUpdateJob(&library.SelfUpdateJobArgs{
		LinuxImageResource:      linuxImage,
		ConcourseBuilderGit:     concourseBuilderGit,
		ImageRegistry:           imageRegistry,
		GoImage:                 goImage,
		ResourceRegistry:        mainPipeline.ResourceRegistry,
		Concourse:               concourse,
		Environment:             environment,
		GenerateProjectLocation: generateProjectLocation,
	})

	branchesJob := BranchesJob(&BranchesJobArgs{
		ConcourseBuilderGit:     concourseBuilderGit,
		ImageRegistry:           imageRegistry,
		GoImage:                 goImage,
		ResourceRegistry:        mainPipeline.ResourceRegistry,
		Concourse:               concourse,
		TargetGitRepo:           targetGit,
		Environment:             environment,
		GenerateProjectLocation: generateProjectLocation,
	})

	mainPipeline.Jobs = project.Jobs{
		selfUpdateJob,
		branchesJob,
	}

	prj.Pipelines = append(prj.Pipelines,
		mainPipeline,
		concourseBuilderPipeline)
	return prj, nil
}
