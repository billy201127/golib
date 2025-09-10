package main

import (
	"fmt"

	"gomod.pri/golib/confuse"
)

func main() {
	sdk := confuse.NewObfuscatorSDK(100)
	// obfWords := sdk.ObfuscateWord("hello")
	deobfWords := sdk.DeobfuscateWord("meta")
	// fmt.Println(obfWords)
	fmt.Println(deobfWords)
}
