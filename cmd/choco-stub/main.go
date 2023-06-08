package main

import (
	"fmt"
)

func main() {
	hint := `
Dear users,
Please note that we moved to a new Chocolatey registry that can be found at https://chocolatey.kubeshop.io/

Please add this URL to the list of Chocolatey sources and use it to install and upgrade Testkube package.
Installation instruction can be found at: https://docs.testkube.io/articles/step1-installing-cli#macos
`
	fmt.Println(hint)
}
