package image

import (
	"time"

	"github.com/concourse-friends/concourse-builder/model"
	"github.com/concourse-friends/concourse-builder/project"
	"github.com/concourse-friends/concourse-builder/resource"
)

var Couchbase = &project.Resource{
	Name: "couchbase-image",
	Type: resource.ImageResourceType.Name,
	Source: &Source{
		Registry:   DockerHub,
		Repository: "couchbase",
	},
	CheckInterval: model.Duration(24 * time.Hour),
}

var RiakKV = &project.Resource{
	Name: "riak-kv-image",
	Type: resource.ImageResourceType.Name,
	Source: &Source{
		Registry:   DockerHub,
		Repository: "basho/riak-kv",
	},
	CheckInterval: model.Duration(24 * time.Hour),
}
