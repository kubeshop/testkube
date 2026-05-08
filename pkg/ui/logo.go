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
`

func (ui *UI) Logo() {
	fmt.Fprint(ui.Writer, Blue(logo))
	fmt.Fprintln(ui.Writer)
}

func (ui *UI) LogoNoColor() {
	fmt.Fprint(ui.Writer, logo)
	fmt.Fprintln(ui.Writer)
}
