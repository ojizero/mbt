package lib

import (
	"bytes"
	"fmt"
	"os"
	"testing"

	"github.com/mbtproject/mbt/e"
	"github.com/stretchr/testify/assert"
)

type TC struct {
	Template string
	Expected string
}

func TestApplyBranch(t *testing.T) {
	clean()
	repo, err := createTestRepository(".tmp/repo")
	check(t, err)

	check(t, repo.InitModule("app-a"))
	check(t, repo.WriteContent("template.tmpl", `
{{- range $i, $mod := .Modules}}
{{- $mod.Name }},
{{- end}}
`))
	check(t, repo.Commit("first"))

	check(t, repo.SwitchToBranch("feature"))
	check(t, repo.InitModule("app-b"))
	check(t, repo.Commit("second"))

	output := new(bytes.Buffer)
	check(t, ApplyBranch(".tmp/repo", "template.tmpl", "feature", output))

	assert.Equal(t, "app-a,app-b,\n", output.String())
}

func TestApplyCommit(t *testing.T) {
	clean()
	repo, err := createTestRepository(".tmp/repo")
	check(t, err)

	check(t, repo.InitModule("app-a"))
	check(t, repo.WriteContent("template.tmpl", `
{{- range $i, $mod := .Modules}}
{{- $mod.Name }},
{{- end}}
`))
	check(t, repo.Commit("first"))

	check(t, repo.InitModule("app-b"))
	check(t, repo.Commit("second"))

	output := new(bytes.Buffer)
	check(t, ApplyCommit(".tmp/repo", repo.LastCommit.String(), "template.tmpl", output))

	assert.Equal(t, "app-a,app-b,\n", output.String())
}

func TestApplyHead(t *testing.T) {
	clean()
	repo, err := createTestRepository(".tmp/repo")
	check(t, err)

	check(t, repo.InitModule("app-a"))
	check(t, repo.WriteContent("template.tmpl", `
{{- range $i, $mod := .Modules}}
{{- $mod.Name }},
{{- end}}
`))
	check(t, repo.Commit("first"))

	check(t, repo.SwitchToBranch("feature"))
	check(t, repo.InitModule("app-b"))
	check(t, repo.Commit("second"))

	output := new(bytes.Buffer)
	check(t, ApplyHead(".tmp/repo", "template.tmpl", output))

	assert.Equal(t, "app-a,app-b,\n", output.String())
}

func TestApplyLocal(t *testing.T) {
	clean()
	repo, err := createTestRepository(".tmp/repo")
	check(t, err)

	check(t, repo.InitModule("app-a"))
	check(t, repo.WriteContent("template.tmpl", `
{{- range $i, $mod := .Modules}}
{{- $mod.Name }},
{{- end}}
`))
	check(t, repo.Commit("first"))

	check(t, repo.InitModule("app-b"))
	check(t, repo.Commit("second"))

	output := new(bytes.Buffer)
	check(t, ApplyLocal(".tmp/repo", "template.tmpl", output))

	assert.Equal(t, "app-a,app-b,\n", output.String())
}

func TestIncorrectTemplatePath(t *testing.T) {
	clean()
	repo, err := createTestRepository(".tmp/repo")
	check(t, err)

	check(t, repo.InitModule("app-a"))
	check(t, repo.WriteContent("template.tmpl", `
{{- range $i, $mod := .Modules}}
{{- $mod.Name }},
{{- end}}
`))
	check(t, repo.Commit("first"))

	output := new(bytes.Buffer)
	err = ApplyCommit(".tmp/repo", repo.LastCommit.String(), "foo/template.tmpl", output)

	assert.EqualError(t, err, fmt.Sprintf(msgTemplateNotFound, "foo/template.tmpl", repo.LastCommit.String()))
	assert.Equal(t, ErrClassUser, (err.(*e.E)).Class())
	assert.Equal(t, "", output.String())
}

func TestBadTemplate(t *testing.T) {
	clean()
	repo, err := createTestRepository(".tmp/repo")
	check(t, err)

	check(t, repo.InitModule("app-a"))
	check(t, repo.WriteContent("template.tmpl", `
{{- range $i, $mod := .Modules}}
`))
	check(t, repo.Commit("first"))

	output := new(bytes.Buffer)
	err = ApplyCommit(".tmp/repo", repo.LastCommit.String(), "template.tmpl", output)

	assert.EqualError(t, err, msgFailedTemplateParse)
	assert.EqualError(t, (err.(*e.E)).InnerError(), "template: template:2: unexpected EOF")
	assert.Equal(t, ErrClassUser, (err.(*e.E)).Class())
	assert.Equal(t, "", output.String())
}

func TestEnvironmentVariables(t *testing.T) {
	clean()
	repo, err := createTestRepository(".tmp/repo")
	check(t, err)

	check(t, repo.InitModule("app-a"))
	check(t, repo.WriteContent("template.tmpl", "{{.Env.EXTERNAL_VALUE}}"))
	check(t, repo.Commit("first"))

	os.Setenv("EXTERNAL_VALUE", "FOO")

	output := new(bytes.Buffer)
	check(t, ApplyCommit(".tmp/repo", repo.LastCommit.String(), "template.tmpl", output))

	assert.Equal(t, "FOO", output.String())
}

func TestCustomTemplateFuncs(t *testing.T) {
	clean()
	repo, err := createTestRepository(".tmp/repo")
	check(t, err)

	check(t, repo.InitModuleWithOptions("app-a", &Spec{
		Name: "app-a",
		Properties: map[string]interface{}{
			"tags":    []string{"a", "b", "c"},
			"numbers": []int{1, 2, 3},
			"nested": map[string]interface{}{
				"tags": []string{"a", "b", "c"},
			},
			"foo": "bar",
		},
	}))

	cases := []TC{
		TC{Template: `{{- if contains (property (module "app-a") "tags") "a"}}yes{{- end}}`, Expected: "yes"},
		TC{Template: `{{- if contains (property (module "app-b") "tags") "a"}}yes{{- end}}`, Expected: ""},
		TC{Template: `{{- if contains (property (module "app-a") "dags") "a"}}yes{{- end}}`, Expected: ""},
		TC{Template: `{{- if contains (property (module "app-a") "tags") "d"}}yes{{- end}}`, Expected: ""},
		TC{Template: `{{- if contains (property (module "app-a") "numbers") "1"}}yes{{- end}}`, Expected: ""},
		TC{Template: `{{- if contains (property (module "app-a") "numbers") 1}}yes{{- end}}`, Expected: "yes"},
		TC{Template: `{{- if contains (property (module "app-a") "nested.tags") "a"}}yes{{- end}}`, Expected: "yes"},
		TC{Template: `{{- if contains (property (module "app-a") "nested.bags") "a"}}yes{{- end}}`, Expected: ""},
		TC{Template: `{{- if contains (property (module "app-a") "tags.tags") "a"}}yes{{- end}}`, Expected: ""},
		TC{Template: `{{- if contains (property (module "app-a") "nested.") "a"}}yes{{- end}}`, Expected: ""},
		TC{Template: `{{- if contains (property (module "app-a") "nested") "a"}}yes{{- end}}`, Expected: ""},
		TC{Template: `{{- propertyOr (module "app-a") "foo" "car"}}`, Expected: "bar"},
		TC{Template: `{{- propertyOr (module "app-a") "foo.bar" "car"}}`, Expected: "car"},
		TC{Template: `{{- propertyOr (module "app-b") "foo" "car"}}`, Expected: "car"},
	}

	for _, c := range cases {
		check(t, repo.WriteContent("template.tmpl", c.Template))
		check(t, repo.Commit("Update"))

		output := new(bytes.Buffer)
		err = ApplyCommit(".tmp/repo", repo.LastCommit.String(), "template.tmpl", output)
		check(t, err)

		assert.Equal(t, c.Expected, output.String(), "Failed test case %s", c.Template)
	}
}
