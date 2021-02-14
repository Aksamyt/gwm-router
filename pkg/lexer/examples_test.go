package lexer

import "fmt"

func Example() {
	items := Lex("/hello/{name}")
	for item := range items {
		fmt.Printf("%#v\t«%v»\n", item, item)
	}
	//Output:
	// lexer.Item{Typ:1, Val:"/", Pos:0}	«/»
	// lexer.Item{Typ:10, Val:"hello", Pos:1}	«"hello"»
	// lexer.Item{Typ:1, Val:"/", Pos:6}	«/»
	// lexer.Item{Typ:2, Val:"{", Pos:7}	«{»
	// lexer.Item{Typ:11, Val:"name", Pos:8}	«'name'»
	// lexer.Item{Typ:3, Val:"}", Pos:12}	«}»
	// lexer.Item{Typ:12, Val:"", Pos:13}	«EOF»
}

func ExampleItem_String() {
	// All other items just print their value.
	fmt.Println(Item{ItemError, "I am an error", 0})
	fmt.Println(Item{ItemRaw, "path-part", 0})
	fmt.Println(Item{ItemVar, "variable", 0})
	fmt.Println(Item{ItemEOF, "", 0})
	//Output:
	// ERROR I am an error
	// "path-part"
	// 'variable'
	// EOF
}
