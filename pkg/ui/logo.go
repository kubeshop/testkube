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

func (ui *UI) Logo() {
	fmt.Fprint(ui.Writer, Blue(logo))
	fmt.Fprintln(ui.Writer)
}

func (ui *UI) LogoNoColor() {
	fmt.Fprint(ui.Writer, logo)
	fmt.Fprintln(ui.Writer)
}
