package ui

import (
	"fmt"

	"github.com/bclicn/color"
)

var logo = `
 _  __     _         _____ _____ ____ _____ 
| |/ /   _| |__   __|_   _| ____/ ___|_   _|
| ' / | | | '_ \ / _ \| | |  _| \___ \ | |  
| . \ |_| | |_) |  __/| | | |___ ___) || |  
|_|\_\__,_|_.__/ \___||_| |_____|____/ |_|  

                       `

func Logo() {
	fmt.Print(color.Blue(logo))
	fmt.Println()
}
