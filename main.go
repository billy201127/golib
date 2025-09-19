package main

import (
	"fmt"

	"gomod.pri/golib/confuse"
)

func main() {
	has := confuse.HasWord("acco")
	fmt.Println(has)
	sdk := confuse.NewObfuscatorSDK(100000)
	obfWords := sdk.DeobfuscateWords([]string{"receiver", "retrieved"})
	fmt.Println(obfWords, has)
}
