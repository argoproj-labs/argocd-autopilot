// This package is used to solve this issue: https://github.com/argoproj/argo-cd/issues/2907
// Once this ticket is closed we will no longer need this hack
package assets

import (
	"github.com/gobuffalo/packr"
)

var (
	BuiltinPolicyCSV string
	ModelConf        string
	SwaggerJSON      string
	BadgeSVG         string
)

func init() {
	var err error
	box := packr.NewBox("../../assets")
	BuiltinPolicyCSV, err = box.FindString("builtin-policy.csv")
	if err != nil {
		panic(err)
	}
	ModelConf, err = box.FindString("model.conf")
	if err != nil {
		panic(err)
	}
	SwaggerJSON, err = box.FindString("swagger.json")
	if err != nil {
		panic(err)
	}
	BadgeSVG, err = box.FindString("badge.svg")
	if err != nil {
		panic(err)
	}
}
