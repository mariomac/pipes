module github.com/mariomac/pipe

go 1.18

require (
	github.com/hashicorp/hcl/v2 v2.11.1
	github.com/sirupsen/logrus v1.8.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/text v0.13.0
)

require (
	github.com/agext/levenshtein v1.2.1 // indirect
	github.com/apparentlymart/go-textseg/v13 v13.0.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/google/go-cmp v0.3.1 // indirect
	github.com/mitchellh/go-wordwrap v0.0.0-20150314170334-ad45545899c7 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/zclconf/go-cty v1.8.0 // indirect
	golang.org/x/sys v0.5.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// TODO: replace when this PR gets merged https://github.com/hashicorp/hcl/pull/521
replace github.com/hashicorp/hcl/v2 v2.11.1 => github.com/mariomac/hcl/v2 v2.11.2-0.20220326231146-b86f46cbcd08
