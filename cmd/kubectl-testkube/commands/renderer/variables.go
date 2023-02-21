package renderer

import (
	"fmt"

	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
	"github.com/kubeshop/testkube/pkg/ui"
)

func RenderVariables(variables testkube.Variables) {
	if len(variables) > 0 {
		ui.NL()
		ui.Warn("  Variables:   ", fmt.Sprintf("%d", len(variables)))
		for _, v := range variables {
			t := ""
			if v.IsSecret() {
				if v.SecretRef != nil {
					t = fmt.Sprintf("[secret:%s key:%s]", v.SecretRef.Name, v.SecretRef.Key)
				}

				if v.Value != "" {
					t = v.Value
				}

				t += " 🔒"
			} else {
				if v.ConfigMapRef != nil {
					t = fmt.Sprintf("[configmap:%s key:%s]", v.ConfigMapRef.Name, v.ConfigMapRef.Key)
				}

				if v.Value != "" {
					t = v.Value
				}
			}

			ui.Info("  -", fmt.Sprintf("%s = %s", v.Name, t))
		}
	}
}
