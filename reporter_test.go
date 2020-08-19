package reporter_test

import (
	"context"
	"testing"

	"github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic/fake"

	"github.com/aerfio/k8s-reporter"
)

func TestNew(t *testing.T) {
	t.Run("should succeed with dynamicClient and GVR supplied", func(t *testing.T) {
		g := gomega.NewWithT(t)
		opts := []reporter.Option{reporter.WithDynamicClient(fake.NewSimpleDynamicClient(runtime.NewScheme())), reporter.WithGVRSchema(schema.GroupVersionResource{})}
		_, err := reporter.New(opts...)
		g.Expect(err).To(gomega.Succeed())
	})
	t.Run("should fail without GVR supplied", func(t *testing.T) {
		g := gomega.NewWithT(t)
		opts := []reporter.Option{reporter.WithDynamicClient(fake.NewSimpleDynamicClient(runtime.NewScheme()))}
		_, err := reporter.New(opts...)
		g.Expect(err).NotTo(gomega.Succeed())
	})
	t.Run("should fail without dynamic client supplied", func(t *testing.T) {
		g := gomega.NewWithT(t)
		opts := []reporter.Option{reporter.WithGVRSchema(schema.GroupVersionResource{})}
		_, err := reporter.New(opts...)
		g.Expect(err).NotTo(gomega.Succeed())
	})
	t.Run("should fail with no options supplied", func(t *testing.T) {
		g := gomega.NewWithT(t)
		_, err := reporter.New()
		g.Expect(err).NotTo(gomega.Succeed())
	})
}

func TestYamlReporter_Get(t *testing.T) {
	type args struct {
		objects   []runtime.Object
		schema    schema.GroupVersionResource
		name      string
		namespace string
	}

	ctx := context.Background()
	getOpts := metav1.GetOptions{}
	tests := []struct {
		name         string
		args         args
		want         string
		wantErr      bool
		errMatcherFn func(err error) bool
	}{
		{
			name: "should get proper resource from all resouces",
			args: args{
				objects: []runtime.Object{
					newUnstructured("group/version", "Pod", "ns-foo", "name-foo"),
					newUnstructured("group/version", "Pod", "ns-foo", "name-foo2"),
					newUnstructured("group2/version", "Deploy", "ns-foo", "name2-foo"),
					newUnstructured("group/version", "TheKind", "ns-foo", "name-bar"),
					newUnstructured("group/version", "Whatever", "ns-foo", "name-baz"),
					newUnstructured("group2/version", "TheKind", "ns-foo", "name2-baz"),
				},
				schema: schema.GroupVersionResource{
					Group:    "group",
					Version:  "version",
					Resource: "pods",
				},
				name:      "name-foo",
				namespace: "ns-foo",
			},
			wantErr: false,
		},
		{
			name: "should error on no resources",
			args: args{
				objects: []runtime.Object{},
				schema: schema.GroupVersionResource{
					Group:    "",
					Version:  "v1",
					Resource: "pods",
				},
				name:      "name-foo",
				namespace: "ns-foo",
			},
			wantErr:      true,
			errMatcherFn: apierrors.IsNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			opts := reporterOptionsWithFakeClientAndGVR(tt.args.schema, tt.args.objects...)
			r, err := reporter.New(opts...)
			g.Expect(err).To(gomega.Succeed())
			resource, err := r.Get(ctx, tt.args.name, tt.args.namespace, getOpts)

			if tt.wantErr {
				g.Expect(err).NotTo(gomega.Succeed())
				g.Expect(tt.errMatcherFn(err)).To(gomega.BeTrue())
				g.Expect(resource).To(gomega.BeEmpty())
			} else {
				g.Expect(err).To(gomega.Succeed())
				g.Expect(resource).To(gomega.ContainSubstring(tt.args.name))
				g.Expect(resource).To(gomega.ContainSubstring(tt.args.namespace))
				g.Expect(resource).To(gomega.ContainSubstring(tt.args.schema.Version))
			}
		})
	}
}

func TestYamlReporter_List(t *testing.T) {
	type args struct {
		objects   []runtime.Object
		schema    schema.GroupVersionResource
		name      string
		namespace string
	}

	ctx := context.Background()
	listOpts := metav1.ListOptions{}

	tests := []struct {
		name           string
		args           args
		want           []string
		expectedNumber int
		wantErr        bool
	}{
		{
			name: "should list yamlified pods in expected number",
			args: args{
				objects: []runtime.Object{
					newUnstructured("group/version", "Pod", "ns-foo", "name-foo"),
					newUnstructured("group/version", "Pod", "ns-foo", "name-foo2"),
					newUnstructured("group2/version", "Deploy", "ns-foo", "name2-foo"),
					newUnstructured("group/version", "TheKind", "ns-foo", "name-bar"),
					newUnstructured("group/version", "Whatever", "ns-foo", "name-baz"),
					newUnstructured("group2/version", "TheKind", "ns-foo", "name2-baz"),
				},
				schema: schema.GroupVersionResource{
					Group:    "group",
					Version:  "version",
					Resource: "pods",
				},
				namespace: "ns-foo",
			},
			expectedNumber: 2,
		},
		{
			name: "should return empty list without error",
			args: args{
				objects: []runtime.Object{
					newUnstructured("group/version", "Pod", "ns-foo", "name-foo"),
					newUnstructured("group/version", "Pod", "ns-foo", "name-foo2"),
				},
				schema: schema.GroupVersionResource{
					Group:    "group",
					Version:  "version",
					Resource: "pods",
				},
				namespace: "ns-foo-other",
			},
			expectedNumber: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			opts := reporterOptionsWithFakeClientAndGVR(tt.args.schema, tt.args.objects...)
			r, err := reporter.New(opts...)
			g.Expect(err).To(gomega.Succeed())
			list, err := r.List(ctx, tt.args.namespace, listOpts)
			g.Expect(err).To(gomega.Succeed())
			g.Expect(list).To(gomega.HaveLen(tt.expectedNumber))
		})
	}
}

func reporterOptionsWithFakeClientAndGVR(schema schema.GroupVersionResource, objects ...runtime.Object) []reporter.Option {
	return []reporter.Option{reporter.WithDynamicClient(fake.NewSimpleDynamicClient(runtime.NewScheme(), objects...)), reporter.WithGVRSchema(schema)}
}

func newUnstructured(apiVersion, kind, namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": apiVersion,
			"kind":       kind,
			"metadata": map[string]interface{}{
				"namespace": namespace,
				"name":      name,
			},
		},
	}
}
