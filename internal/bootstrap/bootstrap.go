package bootstrap

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"sort"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/discovery"
	memory "k8s.io/client-go/discovery/cached"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
)

//go:embed artifacts/*.tmpl
var artifacts embed.FS

type SetupConfig struct {
	Domain string
}

func RunSetup(ctx context.Context, cfg SetupConfig) error {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	kubeConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, nil).ClientConfig()
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	dynClient, _ := dynamic.NewForConfig(kubeConfig)
	discoveryClient, _ := discovery.NewDiscoveryClientForConfig(kubeConfig)
	mapper := restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(discoveryClient))

	entries, err := artifacts.ReadDir("artifacts")
	if err != nil {
		return err
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	for _, entry := range entries {
		fmt.Printf("Applying %s...\n", entry.Name())

		raw, _ := artifacts.ReadFile("artifacts/" + entry.Name())
		tmpl, _ := template.New("manifest").Parse(string(raw))

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, cfg); err != nil {
			return err
		}

		for _, spec := range bytes.Split(buf.Bytes(), []byte("---")) {
			if len(bytes.TrimSpace(spec)) == 0 {
				continue
			}
			if err := applyResource(ctx, dynClient, mapper, spec); err != nil {
				return err
			}
		}
	}
	return nil
}

func applyResource(ctx context.Context, dc dynamic.Interface, mapper meta.RESTMapper, data []byte) error {
	dec := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	obj := &unstructured.Unstructured{}
	_, gvk, err := dec.Decode(data, nil, obj)
	if err != nil {
		return err
	}

	mapping, err := mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	var dr dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		dr = dc.Resource(mapping.Resource).Namespace(obj.GetNamespace())
	} else {
		dr = dc.Resource(mapping.Resource)
	}

	_, err = dr.Apply(ctx, obj.GetName(), obj, metav1.ApplyOptions{
		FieldManager: "deployit-bootstrap",
		Force:        true,
	})
	return err
}
