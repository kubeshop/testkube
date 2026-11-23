package controller

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	aiv1 "github.com/kubeshop/testkube/api/ai/v1alpha1"
	"github.com/kubeshop/testkube/pkg/capabilities"
	"github.com/kubeshop/testkube/pkg/controlplane/declarative"
)

const skillMarkdown = `---
name: k6-load-testing
description: Use when writing a k6 load test.
license: Apache-2.0
allowed-tools: Bash(k6:*) Read
---
# k6

Start small.
`

func skillScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, aiv1.AddToScheme(scheme))
	return scheme
}

func syncedKind(t *testing.T, resource string) declarative.Kind {
	t.Helper()
	for _, k := range SyncedControlPlaneKinds([]capabilities.Capability{capabilities.CapabilityControlPlaneResourcesSkills, capabilities.CapabilityControlPlaneResourcesAI}) {
		if k.Resource == resource {
			return k
		}
	}
	t.Fatalf("%s is not synced", resource)
	return declarative.Kind{}
}

func skillFrom(configMap, key string) *aiv1.SkillSource {
	return &aiv1.SkillSource{ConfigMapKeyRef: &aiv1.ConfigMapKeyReference{Name: configMap, Key: key}}
}

func configMap(name string, data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "testkube"}, Data: data}
}

// Skills are synced only to a Control Plane that advertises them, before the AI Agents that
// refer to them.
func TestSyncedControlPlaneKinds_Skills(t *testing.T) {
	assert.Equal(t, []string{"skills"}, resources(SyncedControlPlaneKinds([]capabilities.Capability{
		capabilities.CapabilityControlPlaneResourcesSkills,
	})))
	assert.Equal(t, []string{"mcpservers", "llmproviders", "skills", "aiagents"}, resources(SyncedControlPlaneKinds([]capabilities.Capability{
		capabilities.CapabilityControlPlaneResourcesAI,
		capabilities.CapabilityControlPlaneResourcesSkills,
	})))
	assert.True(t, readsSkillSources(syncedKind(t, "skills")))
	assert.True(t, readsSkillSources(syncedKind(t, "aiagents")))
	assert.False(t, readsSkillSources(syncedKind(t, "mcpservers")))
}

// A Skill reading its SKILL.md from a ConfigMap in the runner's namespace is sent as if it were
// written inline, since the Control Plane cannot read the cluster.
func TestControlPlaneResourceSyncReconciler_SkillFromConfigMap(t *testing.T) {
	scheme := skillScheme(t)
	skill := &aiv1.Skill{
		ObjectMeta: metav1.ObjectMeta{Name: "k6-load-testing", Namespace: "testkube"},
		Spec:       aiv1.SkillSpec{SkillFrom: skillFrom("skills", "SKILL.md"), Testkube: &aiv1.SkillTestkubeSettings{AlwaysInclude: true}},
	}
	resources := fake.NewClientBuilder().WithScheme(scheme).WithObjects(skill).Build()
	cms := fake.NewClientBuilder().WithScheme(scheme).WithObjects(configMap("skills", map[string]string{"SKILL.md": skillMarkdown})).Build()
	store := &fakeControlPlaneResourceStore{}

	_, err := controlPlaneResourceSyncReconciler(resources, cms, store, syncedKind(t, "skills"), newPausedResources(), nil).Reconcile(t.Context(), request("k6-load-testing"))
	require.NoError(t, err)

	sent, ok := store.Sent.(*aiv1.Skill)
	require.True(t, ok)
	assert.Equal(t, aiv1.SkillSpec{
		Description:  "Use when writing a k6 load test.",
		License:      "Apache-2.0",
		AllowedTools: "Bash(k6:*) Read",
		Content:      "# k6\n\nStart small.\n",
		Testkube:     &aiv1.SkillTestkubeSettings{AlwaysInclude: true},
	}, sent.Spec)
	payload, err := declarative.ApplyPayload(sent)
	require.NoError(t, err)
	assert.NotContains(t, string(payload), "skillFrom")
}

// An AI Agent's own Skill reads its SKILL.md the same way; its skillRefs are sent as they are.
func TestControlPlaneResourceSyncReconciler_AIAgentOwnSkillFromConfigMap(t *testing.T) {
	scheme := skillScheme(t)
	agent := &aiv1.AIAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "load-tester", Namespace: "testkube"},
		Spec: aiv1.AIAgentSpec{
			Instructions: "Test load.",
			Skills:       []aiv1.AIAgentSkill{{Name: "k6-load-testing", SkillFrom: skillFrom("skills", "k6.md")}},
			SkillRefs:    []aiv1.AIAgentSkillRef{{Name: "company-style", Scope: aiv1.SkillScopeOrganization}},
		},
	}
	resources := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent).Build()
	cms := fake.NewClientBuilder().WithScheme(scheme).WithObjects(configMap("skills", map[string]string{"k6.md": skillMarkdown})).Build()
	store := &fakeControlPlaneResourceStore{}

	_, err := controlPlaneResourceSyncReconciler(resources, cms, store, syncedKind(t, "aiagents"), newPausedResources(), nil).Reconcile(t.Context(), request("load-tester"))
	require.NoError(t, err)

	sent := store.Sent.(*aiv1.AIAgent)
	require.Len(t, sent.Spec.Skills, 1)
	assert.Nil(t, sent.Spec.Skills[0].SkillFrom)
	assert.Equal(t, "Use when writing a k6 load test.", sent.Spec.Skills[0].Description)
	assert.Equal(t, "# k6\n\nStart small.\n", sent.Spec.Skills[0].Content)
	assert.Equal(t, []aiv1.AIAgentSkillRef{{Name: "company-style", Scope: aiv1.SkillScopeOrganization}}, sent.Spec.SkillRefs)
}

// A ConfigMap or key that is missing may come with the next Git sync, so it is retried; a SKILL.md
// naming another Skill is fixed only by a change, so it is not. Neither is sent.
func TestControlPlaneResourceSyncReconciler_SkillFromErrors(t *testing.T) {
	scheme := skillScheme(t)
	skill := func(name, key string) *aiv1.Skill {
		return &aiv1.Skill{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "testkube"}, Spec: aiv1.SkillSpec{SkillFrom: skillFrom("skills", key)}}
	}
	cms := fake.NewClientBuilder().WithScheme(scheme).WithObjects(configMap("skills", map[string]string{"SKILL.md": skillMarkdown})).Build()
	noConfigMaps := fake.NewClientBuilder().WithScheme(scheme).Build()

	withDescription := skill("k6-load-testing", "SKILL.md")
	withDescription.Spec.Description = "Mine."

	for name, tc := range map[string]struct {
		skill    *aiv1.Skill
		cms      client.Reader
		terminal bool
		want     string
	}{
		"missing ConfigMap": {skill("k6-load-testing", "SKILL.md"), noConfigMaps, false, `configmaps "skills" not found`},
		"missing key":       {skill("k6-load-testing", "README.md"), cms, false, `ConfigMap "skills" has no key "README.md"`},
		"another name":      {skill("other", "SKILL.md"), cms, true, "the SKILL.md must be named as the Skill is: other"},
		"inline field":      {withDescription, cms, true, "spec.description: Forbidden: comes from the SKILL.md that skillFrom reads"},
	} {
		t.Run(name, func(t *testing.T) {
			resources := fake.NewClientBuilder().WithScheme(scheme).WithObjects(tc.skill).Build()
			store := &fakeControlPlaneResourceStore{}
			_, err := controlPlaneResourceSyncReconciler(resources, tc.cms, store, syncedKind(t, "skills"), newPausedResources(), nil).Reconcile(t.Context(), request(tc.skill.Name))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
			assert.Equal(t, tc.terminal, errors.Is(err, reconcile.TerminalError(nil)), "terminal: %v", err)
			assert.Nil(t, store.Sent)
		})
	}
}

// A Skill whose SKILL.md it can no longer use stops counting as synced, whether the failure is
// retried or not, as nothing of it is sent.
func TestControlPlaneResourceSyncReconciler_SkillFromFailureNotCountedAsSynced(t *testing.T) {
	scheme := skillScheme(t)
	skill := &aiv1.Skill{ObjectMeta: metav1.ObjectMeta{Name: "k6-load-testing", Namespace: "testkube"}, Spec: aiv1.SkillSpec{SkillFrom: skillFrom("skills", "SKILL.md")}}
	resources := fake.NewClientBuilder().WithScheme(scheme).WithObjects(skill).Build()
	renamed := strings.Replace(skillMarkdown, "name: k6-load-testing", "name: other", 1)

	for name, tc := range map[string]struct {
		cms      client.Reader
		terminal bool
	}{
		"SKILL.md names another Skill": {fake.NewClientBuilder().WithScheme(scheme).WithObjects(configMap("skills", map[string]string{"SKILL.md": renamed})).Build(), true},
		"ConfigMap removed":            {fake.NewClientBuilder().WithScheme(scheme).Build(), false},
	} {
		t.Run(name, func(t *testing.T) {
			synced := NewSyncedResources()
			good := fake.NewClientBuilder().WithScheme(scheme).WithObjects(configMap("skills", map[string]string{"SKILL.md": skillMarkdown})).Build()
			_, err := controlPlaneResourceSyncReconciler(resources, good, &fakeControlPlaneResourceStore{}, syncedKind(t, "skills"), newPausedResources(), synced).Reconcile(t.Context(), request("k6-load-testing"))
			require.NoError(t, err)
			require.Equal(t, map[string]int{"skills": 1}, synced.Counts())

			_, err = controlPlaneResourceSyncReconciler(resources, tc.cms, &fakeControlPlaneResourceStore{}, syncedKind(t, "skills"), newPausedResources(), synced).Reconcile(t.Context(), request("k6-load-testing"))
			require.Error(t, err)
			assert.Equal(t, tc.terminal, errors.Is(err, reconcile.TerminalError(nil)), "terminal: %v", err)
			assert.Nil(t, synced.Counts())
		})
	}
}

// A changed ConfigMap syncs again the resources of the kind, in its namespace, that read it.
func TestSkillSourceReaders(t *testing.T) {
	scheme := skillScheme(t)
	skills := []client.Object{
		&aiv1.Skill{ObjectMeta: metav1.ObjectMeta{Name: "k6", Namespace: "testkube"}, Spec: aiv1.SkillSpec{SkillFrom: skillFrom("skills", "k6.md")}},
		&aiv1.Skill{ObjectMeta: metav1.ObjectMeta{Name: "notes", Namespace: "testkube"}, Spec: aiv1.SkillSpec{SkillFrom: skillFrom("skills", "notes.md")}},
		&aiv1.Skill{ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "testkube"}, Spec: aiv1.SkillSpec{SkillFrom: skillFrom("other-skills", "SKILL.md")}},
		&aiv1.Skill{ObjectMeta: metav1.ObjectMeta{Name: "inline", Namespace: "testkube"}, Spec: aiv1.SkillSpec{Description: "D.", Content: "C."}},
		&aiv1.Skill{ObjectMeta: metav1.ObjectMeta{Name: "elsewhere", Namespace: "other"}, Spec: aiv1.SkillSpec{SkillFrom: skillFrom("skills", "SKILL.md")}},
	}
	reader := fake.NewClientBuilder().WithScheme(scheme).WithObjects(skills...).Build()
	list, err := newList(scheme, syncedKind(t, "skills"))
	require.NoError(t, err)

	changed := &metav1.PartialObjectMetadata{ObjectMeta: metav1.ObjectMeta{Name: "skills", Namespace: "testkube"}}
	requests := skillSourceReaders(reader, list)(t.Context(), changed)
	assert.ElementsMatch(t, []reconcile.Request{
		{NamespacedName: types.NamespacedName{Namespace: "testkube", Name: "k6"}},
		{NamespacedName: types.NamespacedName{Namespace: "testkube", Name: "notes"}},
	}, requests)

	_, err = newList(runtime.NewScheme(), syncedKind(t, "skills"))
	require.Error(t, err, "a kind whose list the scheme lacks cannot be watched")
}
