package reporter

import (
	"context"
	"errors"

	"github.com/ghodss/yaml"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

type YamlReporter struct {
	dynamicCli *dynamic.Interface
	gvrSchema  *schema.GroupVersionResource
	resource   dynamic.NamespaceableResourceInterface
}

type Reader interface {
	List(ctx context.Context, namespace string, options metav1.ListOptions) ([]string, error)
	Get(ctx context.Context, name, namespace string, options metav1.GetOptions) (string, error)
}

var _ Reader = &YamlReporter{}

type Option interface {
	apply(*YamlReporter)
}

type dynCliOption struct {
	DynamicCli dynamic.Interface
}

func (d dynCliOption) apply(opts *YamlReporter) {
	opts.dynamicCli = &d.DynamicCli
}

func WithDynamicClient(dynamicCli dynamic.Interface) Option {
	return dynCliOption{DynamicCli: dynamicCli}
}

type gvrOption struct {
	schema schema.GroupVersionResource
}

func (g gvrOption) apply(opts *YamlReporter) {
	opts.gvrSchema = &g.schema
}

func WithGVRSchema(schema schema.GroupVersionResource) Option {
	return gvrOption{schema: schema}
}

// New creates and validates YamlReporter struct
func New(opts ...Option) (YamlReporter, error) {
	instance := &YamlReporter{}

	for _, opt := range opts {
		opt.apply(instance)
	}

	if err := instance.checkConfig(); err != nil {
		return YamlReporter{}, err
	}

	instance.resource = (*instance.dynamicCli).Resource(*instance.gvrSchema)

	return *instance, nil
}

var NoDynamicCliSetError = errors.New("no dynamicCli set, use reporter.WithDynamicClient during initialization")
var NoGroupVersionResourceSetError = errors.New("no GroupVersionResource set, use reporter.WithGVRSchema during initialization")

func (r YamlReporter) checkConfig() error {
	if r.dynamicCli == nil {
		return NoDynamicCliSetError
	} else if r.gvrSchema == nil {
		return NoGroupVersionResourceSetError
	}

	return nil
}

func (r YamlReporter) List(ctx context.Context, namespace string, options metav1.ListOptions) ([]string, error) {
	// context is here for future, when we migrate to k8s libs for v1.18
	unstructuredList, err := r.resource.Namespace(namespace).List(options)
	if err != nil {
		return nil, err
	}

	resources := []string{}

	for _, item := range unstructuredList.Items {
		out, err := yaml.Marshal(item.Object)
		if err != nil {
			return nil, err
		}

		resources = append(resources, string(out))
	}
	return resources, nil
}

func (r YamlReporter) Get(ctx context.Context, name, namespace string, options metav1.GetOptions) (string, error) {
	// context is here for future, when we migrate to k8s libs for v1.18
	unstructuredObj, err := r.resource.Namespace(namespace).Get(name, options)
	if err != nil {
		return "", err
	}

	out, err := yaml.Marshal(unstructuredObj.Object)
	if err != nil {
		return "", err
	}

	return string(out), nil
}
