package ui

import (
	"fmt"

	"github.com/bclicn/color"
)

var logo = `
 __ _  _  _  ____  ____  ____  ____  ____ 
(  / )/ )( \(  _ \(_  _)(  __)/ ___)(_  _)
 )  ( ) \/ ( ) _ (  )(   ) _) \___ \  )(  
(__\_)\____/(____/ (__) (____)(____/ (__) 
`

func Logo() {
	fmt.Print(color.Blue(logo))
	fmt.Println()
}
