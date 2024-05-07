package controller

import (
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FilterHostConfig struct {
}

func (f *FilterHostConfig) ApplyToList(c *client.ListOptions) {
	ls := c.LabelSelector
	if ls == nil {
		ls = labels.NewSelector()
	}

	r, _ := labels.NewRequirement(LabelConfig, selection.Equals, []string{"true"})

	ls = ls.Add(*r)

	c.LabelSelector = ls
}
