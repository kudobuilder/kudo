package cmd

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/thoas/go-funk"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apiextfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	testing2 "k8s.io/client-go/testing"
	fake2 "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/output"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kube"
	"github.com/kudobuilder/kudo/pkg/kudoctl/kudohome"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"
)

var updateGolden = flag.Bool("update", false, "update .golden files and manifests in /config/crd")

type fakeClient struct {
	client *kube.Client
	fc     *fake.Clientset
	fc2    *apiextfake.Clientset
}

func newFakeClient(objs ...runtime.Object) fakeClient {
	fc := fake.NewSimpleClientset(objs...)
	fc2 := apiextfake.NewSimpleClientset()
	cc := fake2.NewFakeClientWithScheme(scheme.Scheme, objs...)

	return fakeClient{
		client: &kube.Client{
			KubeClient: fc,
			ExtClient:  fc2,
			CtrlClient: cc,
		},
		fc:  fc,
		fc2: fc2,
	}
}

func TestInitCmd_exists(t *testing.T) {
	c := newFakeClient(&v1beta1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "kudo-system",
			Name:      "kudo-manager-deploy",
		},
	})

	var buf bytes.Buffer

	cmd := &initCmd{
		out:                 &buf,
		fs:                  afero.NewMemMapFs(),
		client:              c.client,
		image:               "fake",
		selfSignedWebhookCA: true,
	}
	clog.InitWithFlags(nil, &buf)
	Settings.Home = "/opt"

	if err := cmd.run(); err != nil {
		t.Errorf("did not expect error: %v", err)
	}
	expected := "$KUDO_HOME has been configured at /opt\n"

	if !strings.Contains(buf.String(), expected) {
		t.Errorf("expected %q, got %q", expected, buf.String())
	}
}

// TestInitCmd_output tests that init -o can be decoded
func TestInitCmd_output(t *testing.T) {
	c := newFakeClient()

	MockCRD(c.client, "certificates.cert-manager.io", "v1alpha2")
	MockCRD(c.client, "issuers.cert-manager.io", "v1alpha2")

	tests := output.ValidTypes
	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("output %s", tt), func(t *testing.T) {
			var buf bytes.Buffer
			var errOut bytes.Buffer
			cmd := &initCmd{
				out:     &buf,
				errOut:  &errOut,
				client:  c.client,
				output:  tt,
				dryRun:  true,
				version: "dev",
			}
			// ensure that we can marshal
			if err := cmd.run(); err != nil {
				t.Fatal(err)
			}
			// ensure no modifying calls against the server
			forbiddenVerbs := []string{"create", "update", "patch", "delete"}
			for _, a := range c.fc.Actions() {
				if funk.Contains(forbiddenVerbs, a.GetVerb()) {
					t.Errorf("got modifying server call: %v", a)
				}
			}

			assert.True(t, len(buf.Bytes()) > 0, "Buffer needs to have an output")
			// ensure we can decode what was created
			var obj interface{}
			decoder := yamlutil.NewYAMLOrJSONDecoder(&buf, 4096)
			for {
				err := decoder.Decode(&obj)
				if err != nil {
					if err == io.EOF {
						break
					}
					t.Errorf("error decoding init %s output %s %s", tt, err, buf.String())
				}
			}
		})
	}
}

func TestInitCmd_yamlOutput(t *testing.T) {
	customNs := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "foo",
		},
	}
	customSa := &v1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "foo",
			Name:      "safoo",
		},
	}
	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "safoocrb",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Namespace: "foo",
				Name:      "safoo",
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
	}
	c := newFakeClient(crb, customNs, customSa)

	MockCRD(c.client, "certificates.cert-manager.io", "v1alpha2")
	MockCRD(c.client, "issuers.cert-manager.io", "v1alpha2")

	tests := []struct {
		name          string
		goldenFile    string
		flags         map[string]string
		expectedError string
	}{
		{name: "custom namespace", goldenFile: "deploy-kudo-ns.yaml", flags: map[string]string{"dry-run": "true", "output": "yaml", "namespace": "foo", "version": "dev"}},
		{name: "yaml output", goldenFile: "deploy-kudo.yaml", flags: map[string]string{"dry-run": "true", "output": "yaml", "version": "dev"}},
		{name: "service account", goldenFile: "deploy-kudo-sa.yaml", flags: map[string]string{"dry-run": "true", "output": "yaml", "service-account": "safoo", "namespace": "foo", "version": "dev"}},
		{name: "json output", goldenFile: "deploy-kudo.json", flags: map[string]string{"dry-run": "true", "output": "json", "version": "dev"}},
		{name: "invalid output", expectedError: output.InvalidOutputError, flags: map[string]string{"dry-run": "true", "output": "invalid", "version": "dev"}},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			out := &bytes.Buffer{}
			errOut := &bytes.Buffer{}
			initCmd := newInitCmd(fs, out, errOut, c.client)

			Settings.AddFlags(initCmd.Flags())

			for f, value := range tt.flags {
				if err := initCmd.Flags().Set(f, value); err != nil {
					t.Fatal(err)
				}
			}

			if err := initCmd.RunE(initCmd, []string{}); err != nil {
				if tt.expectedError != "" {
					assert.Equal(t, tt.expectedError, err.Error())
				} else {
					t.Fatal(err, errOut.String())
				}
			}

			if tt.goldenFile != "" {
				gp := filepath.Join("testdata", tt.goldenFile+".golden")

				if *updateGolden {
					t.Logf("updating golden file %s", tt.goldenFile)

					//nolint:gosec
					if err := ioutil.WriteFile(gp, out.Bytes(), 0644); err != nil {
						t.Fatalf("failed to update golden file: %s", err)
					}
				}
				g, err := ioutil.ReadFile(gp)
				if err != nil {
					t.Fatalf("failed reading .golden: %s", err)
				}

				assert.Equal(t, string(g), out.String(), "for golden file: %s, for test %s", gp, tt.name)
			}
		})
	}

}

func TestNewInitCmd(t *testing.T) {
	fs := afero.NewMemMapFs()
	var tests = []struct {
		name         string
		flags        map[string]string
		parameters   []string
		errorMessage string
	}{
		{name: "arguments invalid", parameters: []string{"foo"}, errorMessage: "this command does not accept arguments"},
		{name: "name and version together invalid", flags: map[string]string{"kudo-image": "foo", "version": "bar"}, errorMessage: "specify either 'kudo-image' or 'version', not both"},
		{name: "crd-only and wait together invalid: there's nothing to wait for", flags: map[string]string{"crd-only": "true", "wait": "true"}, errorMessage: "wait is not allowed with crd-only"},
		{name: "wait-timeout invalid without wait", flags: map[string]string{"wait-timeout": "400"}, errorMessage: "wait-timeout is only useful when using the flag '--wait'"},
		{name: "crd-only and upgrade together invalid", flags: map[string]string{"crd-only": "true", "upgrade": "true"}, errorMessage: "'--upgrade' and '--crd-only' can not be used at the same time: you can not upgrade *only* crds"},
		{name: "verify and dry-run together invalid: verify is read-only anyway", flags: map[string]string{"dry-run": "true", "verify": "true"}, errorMessage: "'--dry-run' and '--verify' can not be used at the same time"},
		{name: "verify and upgrade together invalid: verify is a separate command", flags: map[string]string{"upgrade": "true", "verify": "true"}, errorMessage: "'--upgrade' and '--verify' can not be used at the same time"},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			out := &bytes.Buffer{}
			errOut := &bytes.Buffer{}
			initCmd := newInitCmd(fs, out, errOut, nil)
			for key, value := range tt.flags {
				if err := initCmd.Flags().Set(key, value); err != nil {
					t.Fatal(err)
				}
			}
			err := initCmd.RunE(initCmd, tt.parameters)
			assert.EqualError(t, err, tt.errorMessage)
		})
	}
}

func TestClientInitialize(t *testing.T) {

	fs := afero.NewMemMapFs()
	home := "kudo_home"
	err := fs.Mkdir(home, 0755)
	if err != nil {
		t.Fatal(err)
	}

	b := bytes.NewBuffer(nil)
	hh := kudohome.Home(home)

	i := &initCmd{fs: fs, out: b, home: hh}
	if err := i.ensureClient(); err != nil {
		t.Error(err)
	}

	expectedDirs := []string{hh.String(), hh.Repository()}
	for _, dir := range expectedDirs {
		if fi, err := fs.Stat(dir); err != nil {
			t.Errorf("%s", err)
		} else if !fi.IsDir() {
			t.Errorf("%s is not a directory", fi)
		}
	}

	if fi, err := fs.Stat(hh.RepositoryFile()); err != nil {
		t.Error(err)
	} else if fi.IsDir() {
		t.Errorf("%s should not be a directory", fi)
	}

	// verifies we can unmarshal file that was created
	// verifies that the default URL is populated
	RepositoryURL := repo.Default.URL
	r, err := repo.LoadRepositories(fs, hh.RepositoryFile())
	if err != nil {
		t.Error(err)
	}
	assert.Equal(t, r.CurrentConfiguration().URL, RepositoryURL)
}

func MockCRD(client *kube.Client, crdName string, apiVersion string) {
	client.ExtClient.(*apiextfake.Clientset).Fake.PrependReactor("get", "customresourcedefinitions", func(action testing2.Action) (handled bool, ret runtime.Object, err error) {

		getAction, _ := action.(testing2.GetAction)
		if getAction != nil {
			if getAction.GetName() == crdName {
				crd := &apiextv1.CustomResourceDefinition{
					TypeMeta: metav1.TypeMeta{
						APIVersion: apiVersion,
					},
					ObjectMeta: metav1.ObjectMeta{
						Name: crdName,
					},
					Spec: apiextv1.CustomResourceDefinitionSpec{
						Versions: []apiextv1.CustomResourceDefinitionVersion{
							{
								Name:    apiVersion,
								Served:  true,
								Storage: true,
							},
						},
					},
					Status: apiextv1.CustomResourceDefinitionStatus{},
				}
				return true, crd, nil
			}
		}

		return false, nil, nil
	})
}
