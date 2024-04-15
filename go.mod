module github.com/mariomac/pipes

go 1.18

require github.com/stretchr/testify v1.7.0

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/kr/pretty v0.1.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/check.v1 v1.0.0-20180628173108-788fd7840127 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// TODO: replace when this PR gets merged https://github.com/hashicorp/hcl/pull/521
replace github.com/hashicorp/hcl/v2 v2.11.1 => github.com/mariomac/hcl/v2 v2.11.2-0.20220326231146-b86f46cbcd08
