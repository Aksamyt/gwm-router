package lexer

import "fmt"

func Example() {
	items := Lex("/hello/{name}")
	for item := range items {
		fmt.Printf("%#v\t«%v»\n", item, item)
	}
	//Output:
	// lexer.Item{Typ:1, Val:"/"}	«/»
	// lexer.Item{Typ:10, Val:"hello"}	«"hello"»
	// lexer.Item{Typ:1, Val:"/"}	«/»
	// lexer.Item{Typ:2, Val:"{"}	«{»
	// lexer.Item{Typ:11, Val:"name"}	«'name'»
	// lexer.Item{Typ:3, Val:"}"}	«}»
	// lexer.Item{Typ:12, Val:""}	«EOF»
}

func ExampleItem_String() {
	// All other items just print their value.
	fmt.Println(Item{ItemError, "I am an error"})
	fmt.Println(Item{ItemRaw, "path-part"})
	fmt.Println(Item{ItemVar, "variable"})
	fmt.Println(Item{ItemEOF, ""})
	//Output:
	// ERROR I am an error
	// "path-part"
	// 'variable'
	// EOF
}
