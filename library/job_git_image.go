package library

import (
	"github.com/concourse-friends/concourse-builder/library/primitive"
	"github.com/concourse-friends/concourse-builder/project"
	"github.com/concourse-friends/concourse-builder/resource"
	"github.com/jinzhu/copier"
)

type GitImageJobArgs struct {
	ConcourseBuilderGitSource *GitSource
	ImageRegistry             *ImageRegistry
	ResourceRegistry          *project.ResourceRegistry
	Tag                       ImageTag
}

func GitImageJob(args *GitImageJobArgs) (*project.Resource, *project.Job) {
	resourceName := project.ResourceName("git-image")
	image := args.ResourceRegistry.GetResource(resourceName)
	if image != nil {
		return image, image.NeededJobs[0]
	}

	curlImageJobArgs := &CurlImageJobArgs{}
	copier.Copy(curlImageJobArgs, args)

	curlImage, _ := CurlImageJob(curlImageJobArgs)

	image = &project.Resource{
		Name: resourceName,
		Type: resource.ImageResourceType.Name,
		Source: &ImageSource{
			Tag:        args.Tag,
			Registry:   args.ImageRegistry,
			Repository: "concourse-builder/git-image",
		},
	}

	dockerSteps := &primitive.Location{
		Volume: &project.JobResource{
			Name:    ConcourseBuilderGitName,
			Trigger: true,
		},
		RelativePath: "docker/git",
	}

	args.ResourceRegistry.MustRegister(UbuntuImage)

	job := BuildImage(
		UbuntuImage,
		curlImage,
		&BuildImageArgs{
			Name:               "git",
			DockerFileResource: dockerSteps,
			Image:              image.Name,
		})

	image.NeededJobs = project.Jobs{job}
	args.ResourceRegistry.MustRegister(image)

	return image, job
}
