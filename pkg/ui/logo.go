package ui

import (
	"fmt"

	"github.com/bclicn/color"
)

var logo = `
██   ██ ██    ██ ██████  ████████ ███████ ███████ ████████ 
██  ██  ██    ██ ██   ██    ██    ██      ██         ██    
█████   ██    ██ ██████     ██    █████   ███████    ██    
██  ██  ██    ██ ██   ██    ██    ██           ██    ██    
██   ██  ██████  ██████     ██    ███████ ███████    ██    
                               /kjuːb tɛst/ by Kubeshop

`

func Logo() {
	fmt.Print(color.Blue(logo))
	fmt.Println()
}

func LogoNoColor() {
	fmt.Print(logo)
	fmt.Println()
}
