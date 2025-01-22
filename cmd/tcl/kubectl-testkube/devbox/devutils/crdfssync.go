// Copyright 2024 Testkube.
//
// Licensed as a Testkube Pro file under the Testkube Community
// License (the "License"); you may not use this file except in compliance with
// the License. You may obtain a copy of the License at
//
//	https://github.com/kubeshop/testkube/blob/main/licenses/TCL.txt

package devutils

import (
	"bytes"
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testworkflowsv1 "github.com/kubeshop/testkube-operator/api/testworkflows/v1"
	"github.com/kubeshop/testkube/internal/crdcommon"
	"github.com/kubeshop/testkube/pkg/testworkflows/executionworker/controller/store"
)

type CRDFSSyncWorkflow struct {
	Workflow   testworkflowsv1.TestWorkflow
	SourcePath string
}

type CRDFSSyncTemplate struct {
	Template   testworkflowsv1.TestWorkflowTemplate
	SourcePath string
}

type CRDFSSyncUpdateOp string

const (
	CRDFSSyncUpdateOpCreate CRDFSSyncUpdateOp = "create"
	CRDFSSyncUpdateOpUpdate CRDFSSyncUpdateOp = "update"
	CRDFSSyncUpdateOpDelete CRDFSSyncUpdateOp = "delete"
)

type CRDFSSyncUpdate struct {
	Template *testworkflowsv1.TestWorkflowTemplate
	Workflow *testworkflowsv1.TestWorkflow
	Op       CRDFSSyncUpdateOp
}

type CRDFSSync struct {
	workflows []CRDFSSyncWorkflow
	templates []CRDFSSyncTemplate
	updates   []CRDFSSyncUpdate
	mu        sync.Mutex
	emitter   store.Update
}

// TODO: optimize for duplicates
func NewCRDFSSync() *CRDFSSync {
	return &CRDFSSync{
		workflows: make([]CRDFSSyncWorkflow, 0),
		templates: make([]CRDFSSyncTemplate, 0),
		updates:   make([]CRDFSSyncUpdate, 0),
		emitter:   store.NewUpdate(),
	}
}

func (c *CRDFSSync) WorkflowsCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.workflows)
}

func (c *CRDFSSync) TemplatesCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.templates)
}

func (c *CRDFSSync) Next(ctx context.Context) (*CRDFSSyncUpdate, error) {
	for {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		c.mu.Lock()
		if len(c.updates) > 0 {
			next := c.updates[0]
			c.updates = c.updates[1:]
			c.mu.Unlock()
			return &next, nil
		}
		ch := c.emitter.Next()
		c.mu.Unlock()
		select {
		case <-ctx.Done():
		case <-ch:
		}
	}
}

func (c *CRDFSSync) processWorkflow(sourcePath string, workflow testworkflowsv1.TestWorkflow) error {
	for i := range c.workflows {
		if c.workflows[i].Workflow.Name == workflow.Name {
			v1, _ := crdcommon.SerializeCRD(c.workflows[i].Workflow, crdcommon.SerializeOptions{
				OmitCreationTimestamp: true,
				CleanMeta:             true,
				Kind:                  "TestWorkflow",
				GroupVersion:          &testworkflowsv1.GroupVersion,
			})
			v2, _ := crdcommon.SerializeCRD(workflow, crdcommon.SerializeOptions{
				OmitCreationTimestamp: true,
				CleanMeta:             true,
				Kind:                  "TestWorkflow",
				GroupVersion:          &testworkflowsv1.GroupVersion,
			})
			c.workflows[i].SourcePath = sourcePath
			if !bytes.Equal(v1, v2) {
				c.workflows[i].Workflow = workflow
				c.updates = append(c.updates, CRDFSSyncUpdate{Workflow: &workflow, Op: CRDFSSyncUpdateOpUpdate})
			}
			return nil
		}
	}
	c.workflows = append(c.workflows, CRDFSSyncWorkflow{SourcePath: sourcePath, Workflow: workflow})
	c.updates = append(c.updates, CRDFSSyncUpdate{Workflow: &workflow, Op: CRDFSSyncUpdateOpCreate})
	return nil
}

func (c *CRDFSSync) processTemplate(sourcePath string, template testworkflowsv1.TestWorkflowTemplate) error {
	for i := range c.templates {
		if c.templates[i].Template.Name == template.Name {
			v1, _ := crdcommon.SerializeCRD(c.templates[i].Template, crdcommon.SerializeOptions{
				OmitCreationTimestamp: true,
				CleanMeta:             true,
				Kind:                  "TestWorkflowTemplate",
				GroupVersion:          &testworkflowsv1.GroupVersion,
			})
			v2, _ := crdcommon.SerializeCRD(template, crdcommon.SerializeOptions{
				OmitCreationTimestamp: true,
				CleanMeta:             true,
				Kind:                  "TestWorkflowTemplate",
				GroupVersion:          &testworkflowsv1.GroupVersion,
			})
			c.templates[i].SourcePath = sourcePath
			if !bytes.Equal(v1, v2) {
				c.templates[i].Template = template
				c.updates = append(c.updates, CRDFSSyncUpdate{Template: &template, Op: CRDFSSyncUpdateOpUpdate})
			}
			return nil
		}
	}
	c.templates = append(c.templates, CRDFSSyncTemplate{SourcePath: sourcePath, Template: template})
	c.updates = append(c.updates, CRDFSSyncUpdate{Template: &template, Op: CRDFSSyncUpdateOpCreate})
	return nil
}

func (c *CRDFSSync) deleteTemplate(name string) {
	for i := 0; i < len(c.templates); i++ {
		if c.templates[i].Template.Name == name {
			c.updates = append(c.updates, CRDFSSyncUpdate{
				Template: &testworkflowsv1.TestWorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: c.templates[i].Template.Name}},
				Op:       CRDFSSyncUpdateOpDelete,
			})
			c.templates = append(c.templates[:i], c.templates[i+1:]...)
			i--
			return
		}
	}
}

func (c *CRDFSSync) deleteWorkflow(name string) {
	for i := 0; i < len(c.workflows); i++ {
		if c.workflows[i].Workflow.Name == name {
			c.updates = append(c.updates, CRDFSSyncUpdate{
				Workflow: &testworkflowsv1.TestWorkflow{ObjectMeta: metav1.ObjectMeta{Name: c.workflows[i].Workflow.Name}},
				Op:       CRDFSSyncUpdateOpDelete,
			})
			c.workflows = append(c.workflows[:i], c.workflows[i+1:]...)
			i--
			return
		}
	}
}

func (c *CRDFSSync) deleteFile(path string) error {
	for i := 0; i < len(c.templates); i++ {
		if c.templates[i].SourcePath == path {
			c.updates = append(c.updates, CRDFSSyncUpdate{
				Template: &testworkflowsv1.TestWorkflowTemplate{ObjectMeta: metav1.ObjectMeta{Name: c.templates[i].Template.Name}},
				Op:       CRDFSSyncUpdateOpDelete,
			})
			c.templates = append(c.templates[:i], c.templates[i+1:]...)
			i--
		}
	}
	for i := 0; i < len(c.workflows); i++ {
		if c.workflows[i].SourcePath == path {
			c.updates = append(c.updates, CRDFSSyncUpdate{
				Workflow: &testworkflowsv1.TestWorkflow{ObjectMeta: metav1.ObjectMeta{Name: c.workflows[i].Workflow.Name}},
				Op:       CRDFSSyncUpdateOpDelete,
			})
			c.workflows = append(c.workflows[:i], c.workflows[i+1:]...)
			i--
		}
	}
	return nil
}

func (c *CRDFSSync) loadFile(path string) error {
	// Ignore non-YAML files
	if !strings.HasSuffix(path, ".yml") && !strings.HasSuffix(path, ".yaml") {
		return nil
	}

	defer c.emitter.Emit()

	// Parse the YAML file
	file, err := os.Open(path)
	if err != nil {
		c.deleteFile(path)
		return nil
	}

	prevTemplates := map[string]struct{}{}
	for i := range c.templates {
		if c.templates[i].SourcePath == path {
			prevTemplates[c.templates[i].Template.Name] = struct{}{}
		}
	}
	prevWorkflows := map[string]struct{}{}
	for i := range c.workflows {
		if c.workflows[i].SourcePath == path {
			prevWorkflows[c.workflows[i].Workflow.Name] = struct{}{}
		}
	}

	// TODO: Handle deleted entries
	decoder := yaml.NewDecoder(file)
	for {
		var obj map[string]interface{}
		err := decoder.Decode(&obj)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			break
		}

		if obj["kind"] == nil || !(obj["kind"].(string) == "TestWorkflow" || obj["kind"].(string) == "TestWorkflowTemplate") {
			continue
		}

		if obj["kind"].(string) == "TestWorkflow" {
			bytes, _ := yaml.Marshal(obj)
			tw := testworkflowsv1.TestWorkflow{}
			err := crdcommon.DeserializeCRD(&tw, bytes)
			if tw.Name == "" {
				continue
			}
			if err != nil {
				continue
			}
			delete(prevWorkflows, tw.Name)
			c.processWorkflow(path, tw)
		} else if obj["kind"].(string) == "TestWorkflowTemplate" {
			bytes, _ := yaml.Marshal(obj)
			tw := testworkflowsv1.TestWorkflowTemplate{}
			err := crdcommon.DeserializeCRD(&tw, bytes)
			if tw.Name == "" {
				continue
			}
			if err != nil {
				continue
			}
			delete(prevTemplates, tw.Name)
			c.processTemplate(path, tw)
		}
	}
	file.Close()

	for t := range prevTemplates {
		c.deleteTemplate(t)
	}
	for t := range prevWorkflows {
		c.deleteWorkflow(t)
	}

	return nil
}

func (c *CRDFSSync) Load(path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	path, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	stat, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !stat.IsDir() {
		return c.loadFile(path)
	}

	return filepath.Walk(path, func(path string, info fs.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		return c.loadFile(path)
	})
}
