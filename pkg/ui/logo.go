package ui

import (
	"fmt"
)

var logo = `
████████ ███████ ███████ ████████ ██   ██ ██    ██ ██████  ███████ 
   ██    ██      ██         ██    ██  ██  ██    ██ ██   ██ ██      
   ██    █████   ███████    ██    █████   ██    ██ ██████  █████   
   ██    ██           ██    ██    ██  ██  ██    ██ ██   ██ ██      
   ██    ███████ ███████    ██    ██   ██  ██████  ██████  ███████ 
                                           /tɛst kjub/ by Kubeshop

`

func Logo() {
	fmt.Fprint(Writer, Blue(logo))
	fmt.Fprintln(Writer)
}

func LogoNoColor() {
	fmt.Fprint(Writer, logo)
	fmt.Fprintln(Writer)
}
